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
// var _ mesh.GossipData = GossipPacket([]byte{})
var _ mesh.GossipData = &GossipPacket{}

// Construct an empty GossipDataManager object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with bufs.
func NewGossipDataManager(selfname mesh.PeerName) *GossipDataManager {
	ret := &GossipDataManager{
		bufs: sync.Map{},
		Self: selfname,
		//Sessions:     sync.Map{},
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

	//val, ok_ := st.bufs.Load(fromPeer.String())
	val, ok_ := gdm.bufs.Load(loadPeer.String())
	if !ok_ {
		return nil, false
	}
	return val.(*BufferWithMutex), true
}

func (gdm *GossipDataManager) StoreBuffer(fromPeer mesh.PeerName, opSide OperationSideAt, buf *BufferWithMutex) {
	//st.bufs.Store(fromPeer.String(), buf)
	storePeer := fromPeer
	if opSide == ServerSide {
		storePeer = math.MaxUint64
	}
	gdm.bufs.Store(storePeer.String(), buf)
}

func (gdm *GossipDataManager) Read(fromPeer mesh.PeerName, opSide OperationSideAt) (result []byte) {
	util.OverlayDebugPrintln("GossipDataManager.Read called: start. fromPeer:", fromPeer)
	//if _, ok := st.Sessions.Load(fromPeer); !ok {
	//	st.Sessions.Store(fromPeer, &GossipSession{
	//		LocalAddress:  &PeerAddress{st.Self},
	//		RemoteAddress: &PeerAddress{fromPeer},
	//		SessMtx:       sync.RWMutex{},
	//		GossipDM:      st,
	//	})
	//}

	var copiedBuf = make([]byte, 0)
	//val, ok2 := st.bufs.Load(fromPeer)
	val, ok2 := gdm.LoadBuffer(fromPeer, opSide)
	if !ok2 {
		panic("no such Stream")
	}
	val.Mtx.Lock()
	//copiedBuf = append(copiedBuf, val.([]byte)...)
	copiedBuf = append(copiedBuf, val.Buf...)
	// clear buffer
	//st.bufs.Store(fromPeer, make([]byte, 0))
	val.Buf = make([]byte, 0)

	//val = val.([]byte)[:0]

	//val2, _ := st.Sessions.Load(fromPeer)
	//bufMtx := val2.(*GossipSession).SessMtx
	//bufMtx.Lock()
	//defer bufMtx.Unlock()

	//retBase := val.([]byte)
	util.OverlayDebugPrintln("GossipDataManager.Read called: before checking length of retBase loop.")
	if len(copiedBuf) == 0 {
		//for len(retBase) == 0 {
		//for storedBuf, _ := st.bufs.Load(fromPeer); len(storedBuf.([]byte)) == 0; storedBuf, _ = st.bufs.Load(fromPeer) {
		for storedBuf, _ := gdm.LoadBuffer(fromPeer, opSide); len(storedBuf.Buf) == 0; storedBuf, _ = gdm.LoadBuffer(fromPeer, opSide) {
			// wait unitl data received
			//bufMtx.Unlock()
			val.Mtx.Unlock()
			time.Sleep(1 * time.Millisecond)
			//bufMtx.Lock()
			val.Mtx.Lock()
		}
		//storedBuf, _ := st.bufs.Load(fromPeer)
		//copiedBuf = append(copiedBuf, storedBuf.([]byte)...)
		//st.bufs.Store(fromPeer, make([]byte, 0))
		storedBuf, _ := gdm.LoadBuffer(fromPeer, opSide)
		copiedBuf = append(copiedBuf, storedBuf.Buf...)
		//st.bufs.Store(fromPeer, make([]byte, 0))
		storedBuf.Buf = make([]byte, 0)
		gdm.StoreBuffer(fromPeer, opSide, storedBuf)
		//storedBuf = storedBuf.([]byte)[:0]
	}
	util.OverlayDebugPrintln("GossipDataManager.Read called: after checking length of retBase loop.")
	val.Mtx.Unlock()
	//ret := make([]byte, len(retBase))
	//copy(ret, retBase)

	//bufMtx.Unlock()
	util.OverlayDebugPrintln("GossipDataManager.Read called: end. fromPeer:", fromPeer, " copiedBuf:", copiedBuf)
	//return ret
	return copiedBuf
}

func (gdm *GossipDataManager) Write(fromPeer mesh.PeerName, opSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.Write called. fromPeer:", fromPeer, " data:", data)

	//if _, ok := st.bufs.Load(fromPeer); !ok {
	//	st.Sessions.Store(fromPeer, &GossipSession{
	//		LocalAddress:  &PeerAddress{st.Self},
	//		RemoteAddress: &PeerAddress{fromPeer},
	//		SessMtx:       sync.RWMutex{},
	//		GossipDM:      st,
	//	})
	//}
	var stBuf []byte
	var stBufMtx *sync.Mutex
	var bufWithMtx *BufferWithMutex
	//if val, ok := st.bufs.Load(fromPeer); ok {
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

	//val2, _ := st.Sessions.Load(fromPeer)
	//bufMtx := val2.(*GossipSession).SessMtx
	//bufMtx.Lock()
	//defer bufMtx.Unlock()

	tmpBuf := make([]byte, 0)
	retBuf := make([]byte, 0)
	tmpBuf = append(tmpBuf, stBuf...)
	tmpBuf = append(tmpBuf, data...)
	retBuf = append(retBuf, tmpBuf...)
	bufWithMtx.Buf = tmpBuf
	gdm.StoreBuffer(fromPeer, opSide, bufWithMtx)
	stBufMtx.Unlock()
	//st.bufs.Store(fromPeer, tmpBuf)

	//st.LastRecvPeer = fromPeer

	return nil
}

func (gdm *GossipDataManager) SendToRemote(dest mesh.PeerName, localOpSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.SendToRemote called. dest:", dest, " data:", data)
	c := make(chan struct{})
	gdm.Peer.Actions <- func() {
		defer close(c)
		if gdm.Peer.Send != nil {
			//p.Send.GossipBroadcast(GossipDM)
			//util.OverlayDebugPrintln("SendToRemote", data)
			//st.Peer.Send.GossipUnicast(st.Peer.Destname, data)
			recvOpSide := ServerSide
			if localOpSide == ServerSide {
				recvOpSide = ClientSide
			}
			sendObj := GossipPacket{
				Buf:          data,
				ReceiverSide: recvOpSide,
			}
			encodedData := sendObj.Encode()[0]
			//gdm.Peer.Send.GossipUnicast(dest, data)
			gdm.Peer.Send.GossipUnicast(dest, encodedData)
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

	//err := p.GossipDataMan.Write(src, opSide, data)
	//if err != nil {
	//	panic(err)
	//}
	//ret, _ := p.GossipDataMan.bufs.Load(src)
	//return GossipPacket(ret.([]byte))
	return nil
}

func (gdm *GossipDataManager) WhenClose(remotePeer mesh.PeerName) {
	// TODO: need to modify according to current impl
	//st.bufs.Delete(remotePeer)

	// do nothing
}

func (gdm *GossipDataManager) NewGossipSessionForClient(remotePeer mesh.PeerName) (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{gdm.Self},
		RemoteAddress: &PeerAddress{remotePeer},
		SessMtx:       sync.RWMutex{},
		GossipDM:      gdm,
		SessionSide:   ClientSide,
	}
	if _, ok := gdm.LoadBuffer(remotePeer, ClientSide); !ok {
		gdm.StoreBuffer(remotePeer, ClientSide, NewBufferWithMutex(make([]byte, 0)))
	}

	return ret, nil
}

func (gdm *GossipDataManager) NewGossipSessionForServer() (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{gdm.Self},
		RemoteAddress: &PeerAddress{math.MaxUint64},
		SessMtx:       sync.RWMutex{},
		GossipDM:      gdm,
		SessionSide:   ServerSide,
	}

	return ret, nil
}
