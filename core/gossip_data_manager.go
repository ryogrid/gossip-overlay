package core

import (
	"bytes"
	"encoding/gob"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math"
	"sync"
	"time"
)

type OperationSideAt int

const (
	ServerSide OperationSideAt = iota
	ClientSide
)

type BufferWithMutex struct {
	Buf []byte
	Mtx *sync.Mutex
}

func NewBufferWithMutex(buf []byte) *BufferWithMutex {
	return &BufferWithMutex{
		Buf: buf,
		Mtx: &sync.Mutex{},
	}
}

type GossipDataManager struct {
	// "mesh.PeerName" -> BufferWithMutex
	bufs         sync.Map
	Self         mesh.PeerName
	Peer         *Peer
	LastRecvPeer mesh.PeerName // for server
}

// GossipPacket implements GossipData.
var _ mesh.GossipData = &GossipPacket{}

// Construct an empty GossipDataManager object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with bufs.
func NewGossipDataManager(selfname mesh.PeerName) *GossipDataManager {
	ret := &GossipDataManager{
		bufs:         sync.Map{},
		Self:         selfname,
		Peer:         nil,
		LastRecvPeer: math.MaxUint64,
	}
	// initialize shared buffer for server side (not used on client side)
	ret.Write(math.MaxUint64, ServerSide, []byte{})
	return ret
}

func (gdm *GossipDataManager) LoadBuffer(fromPeer mesh.PeerName, opSide OperationSideAt) (retBuf *BufferWithMutex, ok bool) {
	loadPeer := fromPeer
	if opSide == ServerSide {
		loadPeer = math.MaxUint64
	}

	val, ok_ := gdm.bufs.Load(loadPeer.String())
	if !ok_ {
		return nil, false
	}
	return val.(*BufferWithMutex), true
}

func (gdm *GossipDataManager) StoreBuffer(fromPeer mesh.PeerName, opSide OperationSideAt, buf *BufferWithMutex) {
	storePeer := fromPeer
	if opSide == ServerSide {
		storePeer = math.MaxUint64
	}
	gdm.bufs.Store(storePeer.String(), buf)
}

func (gdm *GossipDataManager) Read(fromPeer mesh.PeerName, opSide OperationSideAt) (result []byte) {
	util.OverlayDebugPrintln("GossipDataManager.Read called: start. fromPeer:", fromPeer)

	var copiedBuf = make([]byte, 0)
	val, ok2 := gdm.LoadBuffer(fromPeer, opSide)
	if !ok2 {
		panic("no such Stream")
	}
	val.Mtx.Lock()
	copiedBuf = append(copiedBuf, val.Buf...)
	val.Buf = make([]byte, 0)

	util.OverlayDebugPrintln("GossipDataManager.Read called: before checking length of retBase loop.")
	if len(copiedBuf) == 0 {
		for storedBuf, _ := gdm.LoadBuffer(fromPeer, opSide); len(storedBuf.Buf) == 0; storedBuf, _ = gdm.LoadBuffer(fromPeer, opSide) {
			// wait unitl data received
			val.Mtx.Unlock()
			time.Sleep(1 * time.Millisecond)
			val.Mtx.Lock()
		}

		storedBuf, _ := gdm.LoadBuffer(fromPeer, opSide)
		copiedBuf = append(copiedBuf, storedBuf.Buf...)
		storedBuf.Buf = make([]byte, 0)
		gdm.StoreBuffer(fromPeer, opSide, storedBuf)
	}
	util.OverlayDebugPrintln("GossipDataManager.Read called: after checking length of retBase loop.")
	val.Mtx.Unlock()

	util.OverlayDebugPrintln("GossipDataManager.Read called: end. fromPeer:", fromPeer, " copiedBuf:", copiedBuf)
	return copiedBuf
}

func (gdm *GossipDataManager) Write(fromPeer mesh.PeerName, opSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.Write called. fromPeer:", fromPeer, " data:", data)

	var stBuf []byte
	var stBufMtx *sync.Mutex
	var bufWithMtx *BufferWithMutex
	if val, ok := gdm.LoadBuffer(fromPeer, opSide); ok {
		bufWithMtx = val
		stBufMtx = val.Mtx
		stBufMtx.Lock()
		stBuf = val.Buf
	} else {
		stBuf = make([]byte, 0)
		//st.bufs.Store(fromPeer, stBuf)
		bufWithMtx = NewBufferWithMutex(stBuf)
		gdm.StoreBuffer(fromPeer, opSide, bufWithMtx)
		stBufMtx = bufWithMtx.Mtx
		stBufMtx.Lock()
	}

	tmpBuf := make([]byte, 0)
	retBuf := make([]byte, 0)
	tmpBuf = append(tmpBuf, stBuf...)
	tmpBuf = append(tmpBuf, data...)
	retBuf = append(retBuf, tmpBuf...)
	bufWithMtx.Buf = tmpBuf
	gdm.StoreBuffer(fromPeer, opSide, bufWithMtx)
	stBufMtx.Unlock()

	return nil
}

func (gdm *GossipDataManager) SendToRemote(dest mesh.PeerName, localOpSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.SendToRemote called. dest:", dest, " data:", data)
	c := make(chan struct{})
	gdm.Peer.Actions <- func() {
		defer close(c)
		if gdm.Peer.Send != nil {
			recvOpSide := ServerSide
			if localOpSide == ServerSide {
				recvOpSide = ClientSide
			}
			sendObj := GossipPacket{
				Buf:          data,
				ReceiverSide: recvOpSide,
			}
			encodedData := sendObj.Encode()[0]
			for {
				err := gdm.Peer.Send.GossipUnicast(dest, encodedData)
				if err == nil {
					break
				} else {
					// TODO: need to implement timeout
					util.OverlayDebugPrintln("GossipDataManager.SendToRemote: err:", err)
					util.OverlayDebugPrintln("GossipDataManager.SendToRemote: 1sec wait and do retry")
					time.Sleep(1 * time.Second)
				}
			}
		} else {
			gdm.Peer.Logger.Printf("no sender configured; not broadcasting update right now")
		}
	}
	<-c

	return nil
}

// Encode serializes our complete GossipDataManager to a Slice of byte-slices.
// see https://golang.org/pkg/encoding/gob/
func (gdm *GossipDataManager) Encode() [][]byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(gdm.bufs); err != nil {
		panic(err)
	}

	return [][]byte{buf.Bytes()}
}

// Merge merges the other GossipData into this one,
// and returns our resulting, complete GossipDataManager.
func (gdm *GossipDataManager) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return other
}

func (gdm *GossipDataManager) WriteToLocalBuffer(p *Peer, src mesh.PeerName, opSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.MergeComplete called. src:", src, " data:", data)
	if opSide == ServerSide {
		gdm.LastRecvPeer = src
		// server side uses only one buffer
		err := p.GossipDataMan.Write(math.MaxUint64, opSide, data)
		if err != nil {
			panic(err)
		}
	} else if opSide == ClientSide {
		err := p.GossipDataMan.Write(src, opSide, data)
		if err != nil {
			panic(err)
		}
	} else {
		panic("invalid opSide")
	}

	return nil
}

func (gdm *GossipDataManager) WhenClose(remotePeer mesh.PeerName) {
	// TODO: need to modify according to current impl

	// do nothing
}

func (gdm *GossipDataManager) NewGossipSessionForClient(remotePeer mesh.PeerName) (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:       &PeerAddress{gdm.Self},
		RemoteAddresses:    []*PeerAddress{&PeerAddress{remotePeer}},
		RemoteAddressesMtx: &sync.Mutex{},
		SessMtx:            sync.RWMutex{},
		GossipDM:           gdm,
		SessionSide:        ClientSide,
	}
	if _, ok := gdm.LoadBuffer(remotePeer, ClientSide); !ok {
		gdm.StoreBuffer(remotePeer, ClientSide, NewBufferWithMutex(make([]byte, 0)))
	}

	return ret, nil
}

func (gdm *GossipDataManager) NewGossipSessionForServer() (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress: &PeerAddress{gdm.Self},
		//RemoteAddresses: &PeerAddress{math.MaxUint64},
		RemoteAddresses:    make([]*PeerAddress, 0),
		RemoteAddressesMtx: &sync.Mutex{},
		SessMtx:            sync.RWMutex{},
		GossipDM:           gdm,
		SessionSide:        ServerSide,
	}

	return ret, nil
}
