package gossip

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"sync"
	"time"
)

type OperationSideAt int

const (
	ServerSide OperationSideAt = iota
	ClientSide
)

type BufferWithMutex struct {
	Buf [][]byte
	Mtx *sync.Mutex
	//ReadMtx *sync.Mutex
}

func NewBufferWithMutex(buf [][]byte) *BufferWithMutex {
	return &BufferWithMutex{
		Buf: buf,
		Mtx: &sync.Mutex{}, // used by write / read operation
		//ReadMtx: &sync.Mutex{}, // used by read operation only
	}
}

type GossipDataManager struct {
	// "mesh.PeerName" -> BufferWithMutex
	bufs sync.Map
	Self mesh.PeerName
	peer *GossipPeer
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
		peer: nil,
	}

	return ret
}

func (gdm *GossipDataManager) loadBuffer(fromPeer mesh.PeerName, streamID uint16) (retBuf *BufferWithMutex, ok bool) {
	loadPeer := fromPeer

	val, ok_ := gdm.bufs.Load(loadPeer.String() + "-" + fmt.Sprintf("%v", streamID))
	if !ok_ {
		util.OverlayDebugPrintln("GossipDataManager::loadBuffer: no such buffer fromPeer:", fromPeer, " streamID:", streamID)
		return nil, false
	}
	return val.(*BufferWithMutex), true
}

func (gdm *GossipDataManager) storeBuffer(fromPeer mesh.PeerName, streamID uint16, buf *BufferWithMutex) {
	storePeer := fromPeer

	gdm.bufs.Store(storePeer.String()+"-"+fmt.Sprintf("%v", streamID), buf)
}

func (gdm *GossipDataManager) removeBuffer(peerName mesh.PeerName, streamID uint16) {
	gdm.bufs.Delete(peerName.String() + "-" + fmt.Sprintf("%v", streamID))
}

func (gdm *GossipDataManager) read(fromPeer mesh.PeerName, streamID uint16) (result []byte) {
	//util.OverlayDebugPrintln("GossipDataManager.read called: start. fromPeer:", fromPeer, " streamID:", streamID)

	var isDataEmpty = false
	var copiedBuf = make([]byte, 0)
	val, ok2 := gdm.loadBuffer(fromPeer, streamID)
	if !ok2 {
		//panic("no such StreamToNotifySelfInfo")
		util.OverlayDebugPrintln("GossipDataManager::read: no such buffer. fromPeer:", fromPeer, " streamID:", streamID)
		//fmt.Println("GossipDataManager::read: no such buffer. fromPeer:", fromPeer, " streamID:", streamID)
		return nil
		//return make([]byte, 0)
	}
	val.Mtx.Lock()
	if len(val.Buf) != 0 {
		copiedBuf = append(copiedBuf, val.Buf[0]...)
		val.Buf = val.Buf[1:]
		val.Mtx.Unlock()
	} else {
		isDataEmpty = true
		val.Mtx.Unlock()
	}

	//val.ReadMtx.Lock()
	//util.OverlayDebugPrintln("GossipDataManager.read called: before checking length of retBase loop.")
	if isDataEmpty {
		time.Sleep(10 * time.Millisecond)
		//for {
		//	val.Mtx.Lock()
		//	if storedBuf, ok := gdm.loadBuffer(fromPeer, streamID); ok {
		//		// wait unitl data received
		//		//storedBuf.Mtx.Lock()
		//		if len(storedBuf.Buf) > 0 {
		//			// end waiting
		//			copiedBuf = append(copiedBuf, val.Buf[0]...)
		//			val.Buf = val.Buf[1:]
		//			val.Mtx.Unlock()
		//			break
		//		}
		//		val.Mtx.Unlock()
		//		time.Sleep(1 * time.Millisecond)
		//	} else {
		//		util.OverlayDebugPrintln("GossipDataManager.read: waiting end because GossipSession should closed.")
		//		val.Mtx.Unlock()
		//		return nil
		//	}
		//}
	}
	//val.ReadMtx.Unlock()
	//util.OverlayDebugPrintln("GossipDataManager.read called: after checking length of retBase loop.")

	//util.OverlayDebugPrintln("GossipDataManager.read called: end. fromPeer:", fromPeer, " streamID", streamID, " copiedBuf:", copiedBuf)
	return copiedBuf
}

func (gdm *GossipDataManager) write(fromPeer mesh.PeerName, streamID uint16, data []byte) error {
	//util.OverlayDebugPrintln("GossipDataManager.write called. fromPeer:", fromPeer, " streamID", streamID, " data:", data)
	util.OverlayDebugPrintln("GossipDataManager.write called. fromPeer:", fromPeer, " streamID", streamID)

	var stBuf [][]byte
	var stBufMtx *sync.Mutex
	var bufWithMtx *BufferWithMutex
	if val, ok := gdm.loadBuffer(fromPeer, streamID); ok {
		bufWithMtx = val
		stBufMtx = val.Mtx
		stBufMtx.Lock()
		stBuf = val.Buf
	} else {
		util.OverlayDebugPrintln("GossipDataManager::write: no such buffer but create. fromPeer:", fromPeer, " streamID:", streamID)
		stBuf = make([][]byte, 0)
		//st.bufs.Store(fromPeer, stBuf)
		bufWithMtx = NewBufferWithMutex(stBuf)
		gdm.storeBuffer(fromPeer, streamID, bufWithMtx)
		stBufMtx = bufWithMtx.Mtx
		stBufMtx.Lock()
		//return errors.New("no such buffer")
	}

	newBlock := make([]byte, 0)
	//newBlock = append(newBlock, stBuf...)
	newBlock = append(newBlock, data...)
	stBuf = append(stBuf, newBlock)
	bufWithMtx.Buf = stBuf
	gdm.storeBuffer(fromPeer, streamID, bufWithMtx)
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
