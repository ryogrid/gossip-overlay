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
	P                 *Peer
	OriginalServerObj *sctp.Association
	//OriginalServerObjsMtx *sync.Mutex
	Streams       []*sctp.Stream
	StreamsMtx    *sync.Mutex
	gossipSession *GossipSession
	//GossipSessionsMtx *sync.Mutex
}

func NewOverlayServer(p *Peer) (*OverlayServer, error) {
	ret := &OverlayServer{
		P: p,
		// active object is last elem
		OriginalServerObj: nil,
		//OriginalServerObjsMtx: &sync.Mutex{},
		Streams:    make([]*sctp.Stream, 0),
		StreamsMtx: &sync.Mutex{},
		//gossipSession: make([]*GossipSession, 0),
		gossipSession: nil,
		//GossipSessionsMtx:     &sync.Mutex{},
	}

	err := ret.InitInternalServerObj()
	return ret, err

}

func decodeUint64FromBytes(buf []byte) uint64 {
	var ret uint64
	binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &ret)
	return ret
}

func (os *OverlayServer) Accept() (*sctp.Stream, mesh.PeerName, error) {
	//os.OriginalServerObjsMtx.Lock()
	//orgServObj := os.OriginalServerObj[len(os.OriginalServerObj)-1]
	//os.OriginalServerObjsMtx.Unlock()
	//stream, err := orgServObj.AcceptStream()
	stream, err := os.OriginalServerObj.AcceptStream()
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

	util.OverlayDebugPrintln("before read remote PeerName from stream")
	var buf [8]byte
	// read PeerName of remote peer (this is internal protocol of gossip-overlay)
	_, err = stream.Read(buf[:])
	if err != nil {
		fmt.Println("err:", err)
	}
	util.OverlayDebugPrintln("after read remote PeerName from stream buf:", buf)

	decodedName := decodeUint64FromBytes(buf[:])
	remotePeerName := mesh.PeerName(decodedName)
	util.OverlayDebugPrintln("remotePeerName from stream: ", remotePeerName)
	//os.GossipSessionsMtx.Lock()
	//conn := os.gossipSession[len(os.gossipSession)-1]
	//os.GossipSessionsMtx.Unlock()
	//conn.RemoteAddresses.PeerName = remotePeerName

	//os.gossipSession.RemoteAddresses.PeerName = remotePeerName
	os.gossipSession.RemoteAddressesMtx.Lock()
	os.gossipSession.RemoteAddresses = append(os.gossipSession.RemoteAddresses, &PeerAddress{remotePeerName})
	os.gossipSession.RemoteAddressesMtx.Unlock()

	//// setup OriginalServerObj for next stream (Accept call)
	//os.InitInternalServerObj()

	util.OverlayDebugPrintln("end of OverlayServer.Accept")
	return stream, remotePeerName, nil
}

func (os *OverlayServer) InitInternalServerObj() error {
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

	//os.GossipSessionsMtx.Lock()
	//os.gossipSession = append(os.gossipSession, conn)
	//os.GossipSessionsMtx.Unlock()
	os.gossipSession = conn

	//os.OriginalServerObjsMtx.Lock()
	//os.OriginalServerObj = append(os.OriginalServerObj, a)
	//os.OriginalServerObjsMtx.Unlock()
	os.OriginalServerObj = a

	return nil
}

func (os *OverlayServer) Close() error {
	//os.GossipSessionsMtx.Lock()
	//for _, s := range os.gossipSession {
	//	s.Close()
	//}
	//os.gossipSession = make([]*GossipSession, 0)
	//os.GossipSessionsMtx.Unlock()
	//
	//os.OriginalServerObjsMtx.Lock()
	//for _, s := range os.OriginalServerObj {
	//	s.Close()
	//}
	//os.OriginalServerObj = make([]*sctp.Association, 0)
	//os.OriginalServerObjsMtx.Unlock()
	os.gossipSession.Close()
	os.gossipSession = nil
	os.OriginalServerObj.Close()
	os.OriginalServerObj = nil
	os.StreamsMtx.Lock()
	for _, s := range os.Streams {
		s.Close()
	}
	//os.Streams = make([]*sctp.Stream, 0)
	os.Streams = nil
	os.StreamsMtx.Unlock()

	return nil
}
