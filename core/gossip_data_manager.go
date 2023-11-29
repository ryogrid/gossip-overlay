package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
	Buf     []byte
	Mtx     *sync.Mutex
	ReadMtx *sync.Mutex
}

func NewBufferWithMutex(buf []byte) *BufferWithMutex {
	return &BufferWithMutex{
		Buf:     buf,
		Mtx:     &sync.Mutex{}, // used by write / read operation
		ReadMtx: &sync.Mutex{}, // used by read operation only
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
	ret.Write(math.MaxUint64, 0, ServerSide, []byte{})
	return ret
}

func (gdm *GossipDataManager) LoadBuffer(fromPeer mesh.PeerName, streamID uint16, opSide OperationSideAt) (retBuf *BufferWithMutex, ok bool) {
	loadPeer := fromPeer
	if opSide == ServerSide {
		loadPeer = math.MaxUint64
	}

	val, ok_ := gdm.bufs.Load(loadPeer.String() + "-" + fmt.Sprintf("%v", streamID))
	if !ok_ {
		util.OverlayDebugPrintln("GossipDataManager::LoadBuffer: no such buffer fromPeer:", fromPeer, " streamID:", streamID, " opSide:", opSide)
		return nil, false
	}
	return val.(*BufferWithMutex), true
}

func (gdm *GossipDataManager) StoreBuffer(fromPeer mesh.PeerName, streamID uint16, opSide OperationSideAt, buf *BufferWithMutex) {
	storePeer := fromPeer
	if opSide == ServerSide {
		storePeer = math.MaxUint64
	}
	gdm.bufs.Store(storePeer.String()+"-"+fmt.Sprintf("%v", streamID), buf)
}

func (gdm *GossipDataManager) RemoveBuffer(peerName mesh.PeerName, streamID uint16) {
	gdm.bufs.Delete(peerName.String() + "-" + fmt.Sprintf("%v", streamID))
}

func (gdm *GossipDataManager) Read(fromPeer mesh.PeerName, streamID uint16, opSide OperationSideAt) (result []byte) {
	util.OverlayDebugPrintln("GossipDataManager.Read called: start. fromPeer:", fromPeer, " streamID:", streamID, " opSide:", opSide)

	var copiedBuf = make([]byte, 0)
	val, ok2 := gdm.LoadBuffer(fromPeer, streamID, opSide)
	if !ok2 {
		//panic("no such StreamToNotifySelfInfo")
		util.OverlayDebugPrintln("GossipDataManager::Read: no such buffer. fromPeer:", fromPeer, " streamID:", streamID, " opSide:", opSide)
		return nil
		//return make([]byte, 0)
	}
	val.Mtx.Lock()
	copiedBuf = append(copiedBuf, val.Buf...)
	val.Buf = make([]byte, 0)
	val.Mtx.Unlock()

	val.ReadMtx.Lock()
	util.OverlayDebugPrintln("GossipDataManager.Read called: before checking length of retBase loop.")
	if len(copiedBuf) == 0 {
		for {
			if storedBuf, ok := gdm.LoadBuffer(fromPeer, streamID, opSide); ok {
				// wait unitl data received
				//storedBuf.Mtx.Lock()
				if len(storedBuf.Buf) > 0 {
					// end waiting
					//storedBuf.Mtx.Unlock()
					break
				}
				//storedBuf.Mtx.Unlock()
				time.Sleep(1 * time.Millisecond)
			} else {
				util.OverlayDebugPrintln("GossipDataManager.Read: waiting end because GossipSession should closed.")
				val.ReadMtx.Unlock()
				//return make([]byte, 0)
				return nil
			}
		}

		storedBuf, _ := gdm.LoadBuffer(fromPeer, streamID, opSide)
		copiedBuf = append(copiedBuf, storedBuf.Buf...)
		storedBuf.Buf = make([]byte, 0)
		gdm.StoreBuffer(fromPeer, streamID, opSide, storedBuf)
	}
	val.ReadMtx.Unlock()
	util.OverlayDebugPrintln("GossipDataManager.Read called: after checking length of retBase loop.")

	util.OverlayDebugPrintln("GossipDataManager.Read called: end. fromPeer:", fromPeer, " streamID", streamID, " copiedBuf:", copiedBuf)
	return copiedBuf
}

func (gdm *GossipDataManager) Write(fromPeer mesh.PeerName, streamID uint16, opSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.Write called. fromPeer:", fromPeer, " streamID", streamID, " data:", data)

	var stBuf []byte
	var stBufMtx *sync.Mutex
	var bufWithMtx *BufferWithMutex
	if val, ok := gdm.LoadBuffer(fromPeer, streamID, opSide); ok {
		bufWithMtx = val
		stBufMtx = val.Mtx
		stBufMtx.Lock()
		stBuf = val.Buf
	} else {
		util.OverlayDebugPrintln("GossipDataManager::Write: no such buffer but create. fromPeer:", fromPeer, " streamID:", streamID, " opSide:", opSide)
		stBuf = make([]byte, 0)
		//st.bufs.Store(fromPeer, stBuf)
		bufWithMtx = NewBufferWithMutex(stBuf)
		gdm.StoreBuffer(fromPeer, streamID, opSide, bufWithMtx)
		stBufMtx = bufWithMtx.Mtx
		stBufMtx.Lock()
		//return errors.New("no such buffer")
	}

	tmpBuf := make([]byte, 0)
	retBuf := make([]byte, 0)
	tmpBuf = append(tmpBuf, stBuf...)
	tmpBuf = append(tmpBuf, data...)
	retBuf = append(retBuf, tmpBuf...)
	bufWithMtx.Buf = tmpBuf
	gdm.StoreBuffer(fromPeer, streamID, opSide, bufWithMtx)
	stBufMtx.Unlock()

	return nil
}

func (gdm *GossipDataManager) SendToRemote(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)
	c := make(chan struct{})
	gdm.Peer.Actions <- func() {
		defer close(c)
		if gdm.Peer.Send != nil {
			//recvOpSide := ClientSide
			sendObj := GossipPacket{
				Buf:          data,
				ReceiverSide: recvOpSide,
				StreamID:     streamID,
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

func (gdm *GossipDataManager) WriteToLocalBuffer(p *Peer, src mesh.PeerName, streamID uint16, opSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.WriteToLocalBuffer called. src:", src, " streamID", streamID, " data:", data)
	if opSide == ServerSide {
		gdm.LastRecvPeer = src
		// server side uses only one buffer
		err := p.GossipDataMan.Write(math.MaxUint64, streamID, opSide, data)
		if err != nil {
			panic(err)
		}
	} else if opSide == ClientSide {
		err := p.GossipDataMan.Write(src, streamID, opSide, data)
		if err != nil {
			panic(err)
		}
	} else {
		panic("invalid opSide")
	}

	return nil
}

func (gdm *GossipDataManager) WhenClose(remotePeer mesh.PeerName, streamID uint16) error {
	util.OverlayDebugPrintln("GossipDataManager.WhenClose called. remotePeer:", remotePeer, " streamID:", streamID)
	gdm.RemoveBuffer(remotePeer, streamID)

	return nil
}

func (gdm *GossipDataManager) NewGossipSessionForClientToServer(remotePeer mesh.PeerName) (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress: &PeerAddress{gdm.Self},
		//RemoteAddress:      []*PeerAddress{&PeerAddress{remotePeer}},
		RemoteAddress: &PeerAddress{remotePeer},
		//RemoteAddressesMtx: &sync.Mutex{},
		//SessMtx:            sync.RWMutex{},
		GossipDM:          gdm,
		LocalSessionSide:  ClientSide,
		RemoteSessionSide: ServerSide,
		StreamID:          0,
	}
	if _, ok := gdm.LoadBuffer(remotePeer, 0, ClientSide); !ok {
		gdm.StoreBuffer(remotePeer, 0, ClientSide, NewBufferWithMutex(make([]byte, 0)))
	}

	return ret, nil
}

func (gdm *GossipDataManager) NewGossipSessionForClientToClient(remotePeer mesh.PeerName, streamID uint16) (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress: &PeerAddress{gdm.Self},
		//RemoteAddress:      []*PeerAddress{&PeerAddress{remotePeer}},
		RemoteAddress: &PeerAddress{remotePeer},
		//RemoteAddressesMtx: &sync.Mutex{},
		//SessMtx:            sync.RWMutex{},
		GossipDM:          gdm,
		LocalSessionSide:  ClientSide,
		RemoteSessionSide: ClientSide,
		StreamID:          streamID,
	}
	if _, ok := gdm.LoadBuffer(remotePeer, streamID, ClientSide); !ok {
		gdm.StoreBuffer(remotePeer, streamID, ClientSide, NewBufferWithMutex(make([]byte, 0)))
	}

	return ret, nil
}

func (gdm *GossipDataManager) NewGossipSessionForServer() (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{gdm.Self},
		RemoteAddress: &PeerAddress{math.MaxUint64},
		//RemoteAddress:      make([]*PeerAddress, 0),
		//RemoteAddressesMtx: &sync.Mutex{},
		//SessMtx:     sync.RWMutex{},
		GossipDM:          gdm,
		LocalSessionSide:  ServerSide,
		RemoteSessionSide: ClientSide,
		StreamID:          0,
	}

	return ret, nil
}
