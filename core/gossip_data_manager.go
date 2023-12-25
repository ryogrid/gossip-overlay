package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"sync"
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
	bufs sync.Map
	Self mesh.PeerName
	Peer *Peer
}

// GossipPacket implements GossipData.
var _ mesh.GossipData = &GossipPacket{}

// Construct an empty GossipDataManager object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with bufs.
func NewGossipDataManager(selfname mesh.PeerName) *GossipDataManager {
	ret := &GossipDataManager{
		bufs: sync.Map{},
		Self: selfname,
		Peer: nil,
	}

	return ret
}

func (gdm *GossipDataManager) LoadBuffer(fromPeer mesh.PeerName, streamID uint16) (retBuf *BufferWithMutex, ok bool) {
	loadPeer := fromPeer

	val, ok_ := gdm.bufs.Load(loadPeer.String() + "-" + fmt.Sprintf("%v", streamID))
	if !ok_ {
		util.OverlayDebugPrintln("GossipDataManager::LoadBuffer: no such buffer fromPeer:", fromPeer, " streamID:", streamID)
		return nil, false
	}
	return val.(*BufferWithMutex), true
}

func (gdm *GossipDataManager) StoreBuffer(fromPeer mesh.PeerName, streamID uint16, buf *BufferWithMutex) {
	storePeer := fromPeer

	gdm.bufs.Store(storePeer.String()+"-"+fmt.Sprintf("%v", streamID), buf)
}

func (gdm *GossipDataManager) RemoveBuffer(peerName mesh.PeerName, streamID uint16) {
	gdm.bufs.Delete(peerName.String() + "-" + fmt.Sprintf("%v", streamID))
}

func (gdm *GossipDataManager) Read(fromPeer mesh.PeerName, streamID uint16) (result []byte) {
	util.OverlayDebugPrintln("GossipDataManager.Read called: start. fromPeer:", fromPeer, " streamID:", streamID)

	var copiedBuf = make([]byte, 0)
	val, ok2 := gdm.LoadBuffer(fromPeer, streamID)
	if !ok2 {
		//panic("no such StreamToNotifySelfInfo")
		util.OverlayDebugPrintln("GossipDataManager::Read: no such buffer. fromPeer:", fromPeer, " streamID:", streamID)
		return nil
		//return make([]byte, 0)
	}
	val.Mtx.Lock()
	copiedBuf = append(copiedBuf, val.Buf...)
	val.Buf = make([]byte, 0)
	val.Mtx.Unlock()

	//val.ReadMtx.Lock()
	//util.OverlayDebugPrintln("GossipDataManager.Read called: before checking length of retBase loop.")
	//if len(copiedBuf) == 0 {
	//	for {
	//		if storedBuf, ok := gdm.LoadBuffer(fromPeer, streamID, opSide); ok {
	//			// wait unitl data received
	//			//storedBuf.Mtx.Lock()
	//			if len(storedBuf.Buf) > 0 {
	//				// end waiting
	//				//storedBuf.Mtx.Unlock()
	//				break
	//			}
	//			//storedBuf.Mtx.Unlock()
	//			time.Sleep(1 * time.Millisecond)
	//		} else {
	//			util.OverlayDebugPrintln("GossipDataManager.Read: waiting end because GossipSession should closed.")
	//			val.ReadMtx.Unlock()
	//			//return make([]byte, 0)
	//			return nil
	//		}
	//	}
	//
	//	storedBuf, _ := gdm.LoadBuffer(fromPeer, streamID, opSide)
	//	copiedBuf = append(copiedBuf, storedBuf.Buf...)
	//	storedBuf.Buf = make([]byte, 0)
	//	gdm.StoreBuffer(fromPeer, streamID, opSide, storedBuf)
	//}
	//val.ReadMtx.Unlock()
	//util.OverlayDebugPrintln("GossipDataManager.Read called: after checking length of retBase loop.")

	util.OverlayDebugPrintln("GossipDataManager.Read called: end. fromPeer:", fromPeer, " streamID", streamID, " copiedBuf:", copiedBuf)
	return copiedBuf
}

func (gdm *GossipDataManager) Write(fromPeer mesh.PeerName, streamID uint16, data []byte) error {
	util.OverlayDebugPrintln("GossipDataManager.Write called. fromPeer:", fromPeer, " streamID", streamID, " data:", data)

	var stBuf []byte
	var stBufMtx *sync.Mutex
	var bufWithMtx *BufferWithMutex
	if val, ok := gdm.LoadBuffer(fromPeer, streamID); ok {
		bufWithMtx = val
		stBufMtx = val.Mtx
		stBufMtx.Lock()
		stBuf = val.Buf
	} else {
		util.OverlayDebugPrintln("GossipDataManager::Write: no such buffer but create. fromPeer:", fromPeer, " streamID:", streamID)
		stBuf = make([]byte, 0)
		//st.bufs.Store(fromPeer, stBuf)
		bufWithMtx = NewBufferWithMutex(stBuf)
		gdm.StoreBuffer(fromPeer, streamID, bufWithMtx)
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
	gdm.StoreBuffer(fromPeer, streamID, bufWithMtx)
	stBufMtx.Unlock()

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
	if _, ok := gdm.LoadBuffer(remotePeer, streamID); !ok {
		gdm.StoreBuffer(remotePeer, streamID, NewBufferWithMutex(make([]byte, 0)))
	}

	return ret, nil
}
