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
	Streams           []*sctp.Stream
	StreamsMtx        *sync.Mutex
	gossipSession     *GossipSession
}

func NewOverlayServer(p *Peer) (*OverlayServer, error) {
	ret := &OverlayServer{
		P:                 p,
		OriginalServerObj: nil,
		Streams:           make([]*sctp.Stream, 0),
		StreamsMtx:        &sync.Mutex{},
		gossipSession:     nil,
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
	stream, err := os.OriginalServerObj.AcceptStream()
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("accepted a stream")

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

	os.gossipSession.RemoteAddressesMtx.Lock()
	os.gossipSession.RemoteAddresses = append(os.gossipSession.RemoteAddresses, &PeerAddress{remotePeerName})
	os.gossipSession.RemoteAddressesMtx.Unlock()

	util.OverlayDebugPrintln("end of OverlayServer.Accept")
	return stream, remotePeerName, nil
}

func (os *OverlayServer) InitInternalServerObj() error {
	conn, err := os.P.GossipDataMan.NewGossipSessionForServer()
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("created a gossip listener")

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err := sctp.Server(config)
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("created a server")

	os.gossipSession = conn
	os.OriginalServerObj = a

	return nil
}

func (os *OverlayServer) Close() error {
	os.gossipSession.Close()
	os.gossipSession = nil
	os.OriginalServerObj.Close()
	os.OriginalServerObj = nil
	os.StreamsMtx.Lock()
	for _, s := range os.Streams {
		s.Close()
	}
	os.Streams = nil
	os.StreamsMtx.Unlock()

	return nil
}
