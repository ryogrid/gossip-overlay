package core

import (
	"bytes"
	"sync"

	"encoding/gob"

	"github.com/weaveworks/mesh"
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

type GossipDataManager struct {
	// mesh.PeerName -> []byte
	Bufs sync.Map
	Self mesh.PeerName
	// mesh.PeerName -> *GossipSession
	Sessions sync.Map
}

/*
// GossipDataManager implements GossipData.
var _ mesh.GossipData = &GossipDataManager{}
*/

// GossipBytes implements GossipData.
var _ mesh.GossipData = GossipBytes([]byte{})

// Construct an empty GossipDataManager object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with Bufs.
func NewConnectionDataManager(self mesh.PeerName) *GossipDataManager {
	return &GossipDataManager{
		Bufs:     sync.Map{},
		Self:     self,
		Sessions: sync.Map{},
	}
}

func (st *GossipDataManager) Read(fromPeer mesh.PeerName) (result []byte) {
	if _, ok := st.Sessions.Load(fromPeer); !ok {
		st.Sessions.Store(fromPeer, &GossipSession{
			RemoteAddress: &PeerAddress{fromPeer},
			SessMtx:       sync.RWMutex{},
			GossipDM:      st,
		})
	}

	val, ok2 := st.Bufs.Load(fromPeer)
	if !ok2 {
		panic("no such Peer")
	}
	st.Bufs.Store(fromPeer, make([]byte, 0))

	val2, _ := st.Sessions.Load(fromPeer)
	bufMtx := val2.(*GossipSession).SessMtx
	bufMtx.Lock()
	defer bufMtx.Unlock()

	retBase := val.([]byte)
	ret := make([]byte, len(retBase))
	copy(ret, retBase)

	return ret
}

func (st *GossipDataManager) Write(fromPeer mesh.PeerName, data []byte) []byte {
	tmpBuf := make([]byte, 0)
	if _, ok := st.Bufs.Load(fromPeer); !ok {
		st.Sessions.Store(fromPeer, &GossipSession{
			RemoteAddress: &PeerAddress{fromPeer},
			SessMtx:       sync.RWMutex{},
			GossipDM:      st,
		})
	}
	var stBuf []byte
	if val, ok := st.Bufs.Load(fromPeer); ok {
		stBuf = val.([]byte)
	} else {
		stBuf = make([]byte, 0)
		st.Bufs.Store(fromPeer, stBuf)
	}

	val2, _ := st.Sessions.Load(fromPeer)
	bufMtx := val2.(*GossipSession).SessMtx
	bufMtx.Lock()
	defer bufMtx.Unlock()

	tmpBuf = append(tmpBuf, stBuf...)
	tmpBuf = append(tmpBuf, data...)
	st.Bufs.Store(fromPeer, tmpBuf)

	return tmpBuf
}

// Encode serializes our complete GossipDataManager to a Slice of byte-slices.
// see https://golang.org/pkg/encoding/gob/
func (st *GossipDataManager) Encode() [][]byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(st.Bufs); err != nil {
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
// Return a non-nil mesh.GossipData representation of the received Bufs.
func (st *GossipDataManager) MergeReceived(p *Peer, src mesh.PeerName, data []byte) (received mesh.GossipData) {
	p.GossipDataMan.Write(src, data)
	return GossipBytes(data)
}
*/

func (st *GossipDataManager) MergeComplete(p *Peer, src mesh.PeerName, data []byte) (complete mesh.GossipData) {
	p.GossipDataMan.Write(src, data)
	val, _ := p.GossipDataMan.Bufs.Load(src)
	return GossipBytes(val.([]byte))
}

func (st *GossipDataManager) Close(remotePeer mesh.PeerName) {
	st.Bufs.Delete(remotePeer)
	st.Sessions.Delete(remotePeer)
}
