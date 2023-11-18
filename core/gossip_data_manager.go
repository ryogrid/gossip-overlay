package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
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

type GossipDataManager struct {
	// mesh.PeerName -> []byte
	Bufs sync.Map
	Self mesh.PeerName
	// mesh.PeerName -> *GossipSession
	Sessions sync.Map
	Peer     *Peer
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
func NewConnectionDataManager(selfname mesh.PeerName) *GossipDataManager {
	return &GossipDataManager{
		Bufs:     sync.Map{},
		Self:     selfname,
		Sessions: sync.Map{},
		Peer:     nil,
	}
}

func (st *GossipDataManager) Read(fromPeer mesh.PeerName) (result []byte) {
	if _, ok := st.Sessions.Load(fromPeer); !ok {
		st.Sessions.Store(fromPeer, &GossipSession{
			LocalAddress:  &PeerAddress{st.Self},
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
	//defer bufMtx.Unlock()

	retBase := val.([]byte)
	if len(retBase) == 0 {
		for len(retBase) == 0 {
			bufMtx.Unlock()
			time.Sleep(1 * time.Millisecond)
			bufMtx.Lock()
		}
	}
	ret := make([]byte, len(retBase))
	copy(ret, retBase)

	bufMtx.Unlock()
	return ret
}

func (st *GossipDataManager) Write(fromPeer mesh.PeerName, data []byte) []byte {
	tmpBuf := make([]byte, 0)
	if _, ok := st.Bufs.Load(fromPeer); !ok {
		st.Sessions.Store(fromPeer, &GossipSession{
			LocalAddress:  &PeerAddress{st.Self},
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

func (st *GossipDataManager) WriteToRemote(data []byte) error {
	c := make(chan struct{})
	st.Peer.Actions <- func() {
		defer close(c)
		if st.Peer.Send != nil {
			//p.Send.GossipBroadcast(GossipDM)
			fmt.Println("WriteToRemote", data)
			st.Peer.Send.GossipUnicast(st.Peer.Destname, data)
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

func (st *GossipDataManager) NewGossipSessionForClient(remotePeer mesh.PeerName) (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{st.Self},
		RemoteAddress: &PeerAddress{remotePeer},
		SessMtx:       sync.RWMutex{},
		GossipDM:      st,
	}
	st.Sessions.Store(remotePeer, ret)
	st.Bufs.Store(remotePeer, make([]byte, 0))

	return ret, nil
}

func (st *GossipDataManager) NewGossipSessionForServer() (*GossipSession, error) {
	ret := &GossipSession{
		LocalAddress:  &PeerAddress{st.Self},
		RemoteAddress: &PeerAddress{math.MaxUint64},
		SessMtx:       sync.RWMutex{},
		GossipDM:      st,
	}
	st.Sessions.Store(mesh.PeerName(math.MaxUint64), ret)
	st.Bufs.Store(mesh.PeerName(math.MaxUint64), make([]byte, 0))
	// store of session is not needed here when creation for server

	return ret, nil
}
