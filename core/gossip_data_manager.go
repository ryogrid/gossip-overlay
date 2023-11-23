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

type GossipBytes []byte

func (buf GossipBytes) Encode() [][]byte {
	var buf2 bytes.Buffer
	if err := gob.NewEncoder(&buf2).Encode(buf); err != nil {
		panic(err)
	}

	return [][]byte{buf2.Bytes()}
}

func (buf GossipBytes) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	retBuf := make([]byte, 0)
	retBuf = append(retBuf, []byte(buf)...)
	retBuf = append(retBuf, []byte(other.(GossipBytes))...)

	return GossipBytes(retBuf)
}

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
	//// mesh.PeerName -> []byte
	// "mesh.PeerName-StreamID" -> BufferWithMutex
	bufs sync.Map
	Self mesh.PeerName
	//// mesh.PeerName -> *GossipSession
	//Sessions     sync.Map
	Peer *Peer
	//LastRecvPeer mesh.PeerName // for server
}

/*
// GossipDataManager implements GossipData.
var _ mesh.GossipData = &GossipDataManager{}
*/

// GossipBytes implements GossipData.
var _ mesh.GossipData = GossipBytes([]byte{})

// Construct an empty GossipDataManager object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with bufs.
func NewConnectionDataManager(selfname mesh.PeerName) *GossipDataManager {
	return &GossipDataManager{
		bufs: sync.Map{},
		Self: selfname,
		//Sessions:     sync.Map{},
		Peer: nil,
		//LastRecvPeer: math.MaxUint64,
	}
}

func (st *GossipDataManager) LoadBuffer(fromPeer mesh.PeerName, streamID uint16) (retBuf *BufferWithMutex, ok bool) {
	val, ok_ := st.bufs.Load(fromPeer.String() + "-" + fmt.Sprint(streamID))
	if !ok_ {
		return nil, false
	}
	return val.(*BufferWithMutex), true
}

func (st *GossipDataManager) StoreBuffer(fromPeer mesh.PeerName, streamID uint16, buf *BufferWithMutex) {
	st.bufs.Store(fromPeer.String()+"-"+fmt.Sprint(streamID), buf)
}

func (st *GossipDataManager) Read(fromPeer mesh.PeerName, streamID uint16) (result []byte) {
	// TODO: decide buffer with fromPeer and streamID

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
	val, ok2 := st.LoadBuffer(fromPeer, streamID)
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
		for storedBuf, _ := st.LoadBuffer(fromPeer, streamID); len(storedBuf.Buf) == 0; storedBuf, _ = st.LoadBuffer(fromPeer, streamID) {
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
		storedBuf, _ := st.LoadBuffer(fromPeer, streamID)
		copiedBuf = append(copiedBuf, storedBuf.Buf...)
		//st.bufs.Store(fromPeer, make([]byte, 0))
		storedBuf.Buf = make([]byte, 0)
		st.StoreBuffer(fromPeer, streamID, storedBuf)
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

func (st *GossipDataManager) Write(fromPeer mesh.PeerName, streamId uint16, data []byte) []byte {
	util.OverlayDebugPrintln("GossipDataManager.Write called. fromPeer:", fromPeer, " streamId:", streamId, " data:", data)

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
	if val, ok := st.LoadBuffer(fromPeer, streamId); ok {
		bufWithMtx = val
		stBufMtx = val.Mtx
		stBufMtx.Lock()
		stBuf = val.Buf
	} else {
		stBuf = make([]byte, 0)
		//st.bufs.Store(fromPeer, stBuf)
		bufWithMtx = NewBufferWithMutex(stBuf)
		st.StoreBuffer(fromPeer, streamId, bufWithMtx)
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
	st.StoreBuffer(fromPeer, streamId, bufWithMtx)
	stBufMtx.Unlock()
	//st.bufs.Store(fromPeer, tmpBuf)

	//// TODO: temporal impl for server side (not work for multi connection)
	//st.LastRecvPeer = fromPeer

	return tmpBuf
}

func (st *GossipDataManager) WriteToRemote(dest mesh.PeerName, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.WriteToRemote called. dest:", dest, " data:", data)
	c := make(chan struct{})
	st.Peer.Actions <- func() {
		defer close(c)
		if st.Peer.Send != nil {
			//p.Send.GossipBroadcast(GossipDM)
			//util.OverlayDebugPrintln("WriteToRemote", data)
			//st.Peer.Send.GossipUnicast(st.Peer.Destname, data)
			st.Peer.Send.GossipUnicast(dest, data)
		} else {
			st.Peer.Logger.Printf("no sender configured; not broadcasting update right now")
		}
	}
	<-c

	return nil
}

// Encode serializes our complete GossipDataManager to a Slice of byte-slices.
// see https://golang.org/pkg/encoding/gob/
func (st *GossipDataManager) Encode() [][]byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(st.bufs); err != nil {
		panic(err)
	}

	return [][]byte{buf.Bytes()}
}

// Merge merges the other GossipData into this one,
// and returns our resulting, complete GossipDataManager.
func (st *GossipDataManager) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return other
}

/*
// Merge the data into our GossipDataManager
// Return a non-nil mesh.GossipData representation of the received bufs.
func (st *GossipDataManager) MergeReceived(p *Peer, src mesh.PeerName, data []byte) (received mesh.GossipData) {
	p.GossipDataMan.Write(src, data)
	return GossipBytes(data)
}
*/

func (st *GossipDataManager) MergeComplete(p *Peer, src mesh.PeerName, data []byte) (complete mesh.GossipData) {
	util.OverlayDebugPrintln("GossipDataManager.MergeComplete called. src:", src, " data:", data)
	ret := p.GossipDataMan.Write(src, data)
	//ret, _ := p.GossipDataMan.bufs.Load(src)
	//return GossipBytes(ret.([]byte))
	return GossipBytes(ret)
}

func (st *GossipDataManager) Close(remotePeer mesh.PeerName) {
	st.bufs.Delete(remotePeer)
	//st.Sessions.Delete(remotePeer)
}

func (st *GossipDataManager) NewGossipSessionForClient(remotePeer mesh.PeerName) (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{st.Self},
		RemoteAddress: &PeerAddress{remotePeer},
		SessMtx:       sync.RWMutex{},
		GossipDM:      st,
	}
	//st.Sessions.Store(remotePeer, ret)
	st.bufs.Store(remotePeer, make([]byte, 0))

	return ret, nil
}

func (st *GossipDataManager) NewGossipSessionForServer() (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{st.Self},
		RemoteAddress: &PeerAddress{math.MaxUint64},
		SessMtx:       sync.RWMutex{},
		GossipDM:      st,
	}
	//st.Sessions.Store(mesh.PeerName(math.MaxUint64), ret)
	st.bufs.Store(mesh.PeerName(math.MaxUint64), make([]byte, 0))
	// store of session is not needed here when creation for server

	return ret, nil
}
