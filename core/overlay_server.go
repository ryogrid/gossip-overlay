package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"log"
	"sync"
)

// wrapper of sctp.Server
type OverlayServer struct {
	P                     *Peer
	OriginalServerObjs    []*sctp.Association
	OriginalServerObjsMtx *sync.Mutex
	Streams               []*sctp.Stream
	StreamsMtx            *sync.Mutex
	GossipSessions        []*GossipSession
	GossipSessionsMtx     *sync.Mutex
}

func NewOverlayServer(p *Peer) (*OverlayServer, error) {
	ret := &OverlayServer{
		P: p,
		// active object is last elem
		OriginalServerObjs:    make([]*sctp.Association, 0),
		OriginalServerObjsMtx: &sync.Mutex{},
		Streams:               make([]*sctp.Stream, 0),
		StreamsMtx:            &sync.Mutex{},
		GossipSessions:        make([]*GossipSession, 0),
		GossipSessionsMtx:     &sync.Mutex{},
	}

	err := ret.PrepareNewServerObj()
	return ret, err

}

func decodeUint64FromBytes(buf []byte) uint64 {
	var ret uint64
	binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &ret)
	return ret
}

func (os *OverlayServer) Accept() (*sctp.Stream, error) {
	os.OriginalServerObjsMtx.Lock()
	defer os.OriginalServerObjsMtx.Unlock()

	stream, err := os.OriginalServerObjs[0].AcceptStream()
	if err != nil {
		log.Panic(err)
	}
	//defer stream.Close()
	util.OverlayDebugPrintln("accepted a stream")

	// set unordered = true and 10ms treshold for dropping packets
	//stream.SetReliabilityParams(true, sctp.ReliabilityTypeTimed, 10)
	stream.SetReliabilityParams(true, sctp.ReliabilityTypeReliable, 0)

	os.StreamsMtx.Lock()
	os.Streams = append(os.Streams, stream)
	os.StreamsMtx.Unlock()

	var buf [8]byte
	// read PeerName of remote peer (this is internal protocol of gossip-overlay)
	_, err = stream.Read(buf[:])
	if err != nil {
		fmt.Println("err:", err)
	}

	decodedName := decodeUint64FromBytes(buf[:])
	remotePeerName := mesh.PeerName(decodedName)
	os.GossipSessionsMtx.Lock()
	conn := os.GossipSessions[len(os.GossipSessions)-1]
	os.GossipSessionsMtx.Unlock()
	conn.RemoteAddress.PeerName = remotePeerName

	// setup OriginalServerObj for next stream (Accept call)
	os.PrepareNewServerObj()

	return stream, nil
}

func (os *OverlayServer) PrepareNewServerObj() error {
	//conn, err := net.ListenUDP("udp", &addr)
	conn, err := os.P.GossipDataMan.NewGossipSessionForServer()
	if err != nil {
		log.Panic(err)
	}
	//defer conn.Close()
	util.OverlayDebugPrintln("created a gossip listener")

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err := sctp.Server(config)
	if err != nil {
		log.Panic(err)
	}
	//defer a.Close()
	util.OverlayDebugPrintln("created a server")

	os.GossipSessionsMtx.Lock()
	os.GossipSessions = append(os.GossipSessions, conn)
	os.GossipSessionsMtx.Unlock()

	os.OriginalServerObjsMtx.Lock()
	os.OriginalServerObjs = append(os.OriginalServerObjs, a)
	os.OriginalServerObjsMtx.Unlock()

	return nil
}

func (os *OverlayServer) Close() error {
	os.GossipSessionsMtx.Lock()
	for _, s := range os.GossipSessions {
		s.Close()
	}
	os.GossipSessions = make([]*GossipSession, 0)
	os.GossipSessionsMtx.Unlock()

	os.OriginalServerObjsMtx.Lock()
	for _, s := range os.OriginalServerObjs {
		s.Close()
	}
	os.OriginalServerObjs = make([]*sctp.Association, 0)
	os.OriginalServerObjsMtx.Unlock()

	os.StreamsMtx.Lock()
	for _, s := range os.Streams {
		s.Close()
	}
	os.Streams = make([]*sctp.Stream, 0)
	os.StreamsMtx.Unlock()

	return nil
}
