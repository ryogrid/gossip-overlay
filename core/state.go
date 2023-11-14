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

// represents a connection from a Peer
type Conn struct {
	PeerName mesh.PeerName
	BufMtx   sync.RWMutex
	// buffered data is accessed through St each time
	St *State
}

type State struct {
	// mesh.PeerName -> []byte
	Bufs sync.Map
	Self mesh.PeerName
	// mesh.PeerName -> *Conn
	Conns sync.Map
}

// State implements GossipData.
var _ mesh.GossipData = &State{}

// GossipBytes implements GossipData.
var _ mesh.GossipData = GossipBytes([]byte{})

// Construct an empty State object, ready to receive updates.
// This is suitable to use at program start.
// Other peers will populate us with Bufs.
func NewState(self mesh.PeerName) *State {
	return &State{
		Bufs:  sync.Map{},
		Self:  self,
		Conns: sync.Map{},
	}
}

func (st *State) Read(fromPeer mesh.PeerName) (result []byte) {
	if _, ok := st.Conns.Load(fromPeer); !ok {
		st.Conns.Store(fromPeer, &Conn{
			PeerName: fromPeer,
			BufMtx:   sync.RWMutex{},
			St:       st,
		})
	}

	val, ok2 := st.Bufs.Load(fromPeer)
	if !ok2 {
		panic("no such Peer")
	}
	st.Bufs.Store(fromPeer, make([]byte, 0))

	val2, _ := st.Conns.Load(fromPeer)
	bufMtx := val2.(*Conn).BufMtx
	bufMtx.Lock()
	defer bufMtx.Unlock()

	retBase := val.([]byte)
	ret := make([]byte, len(retBase))
	copy(ret, retBase)

	return ret
}

func (st *State) Write(fromPeer mesh.PeerName, data []byte) []byte {
	tmpBuf := make([]byte, 0)
	if _, ok := st.Bufs.Load(fromPeer); !ok {
		st.Conns.Store(fromPeer, &Conn{
			PeerName: fromPeer,
			BufMtx:   sync.RWMutex{},
			St:       st,
		})
	}
	var stBuf []byte
	if val, ok := st.Bufs.Load(fromPeer); ok {
		stBuf = val.([]byte)
	} else {
		stBuf = make([]byte, 0)
		st.Bufs.Store(fromPeer, stBuf)
	}

	val2, _ := st.Conns.Load(fromPeer)
	bufMtx := val2.(*Conn).BufMtx
	bufMtx.Lock()
	defer bufMtx.Unlock()

	tmpBuf = append(tmpBuf, stBuf...)
	tmpBuf = append(tmpBuf, data...)
	st.Bufs.Store(fromPeer, tmpBuf)

	return tmpBuf
}

// Encode serializes our complete State to a Slice of byte-slices.
// see https://golang.org/pkg/encoding/gob/
func (st *State) Encode() [][]byte {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(st.Bufs); err != nil {
		panic(err)
	}

	return [][]byte{buf.Bytes()}
}

// Merge merges the other GossipData into this one,
// and returns our resulting, complete State.
func (st *State) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return other
}

// Merge the data into our State
// Return a non-nil mesh.GossipData representation of the received Bufs.
func (st *State) MergeReceived(p *Peer, src mesh.PeerName, data []byte) (received mesh.GossipData) {
	p.St.Write(src, data)
	return GossipBytes(data)
}

func (st *State) MergeComplete(p *Peer, src mesh.PeerName, data []byte) (complete mesh.GossipData) {
	p.St.Write(src, data)
	val, _ := p.St.Bufs.Load(src)
	return GossipBytes(val.([]byte))
}
