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
	"math"
	"sync"
)

// wrapper of sctp.Server
type OverlayServer struct {
	P                 *Peer
	OriginalServerObj *sctp.Association
	ServerStream      *sctp.Stream
	CtoCStreams       []*sctp.Stream
	StreamsMtx        *sync.Mutex
	gossipSession     *GossipSession
}

func NewOverlayServer(p *Peer) (*OverlayServer, error) {
	ret := &OverlayServer{
		P:                 p,
		OriginalServerObj: nil,
		ServerStream:      nil,
		CtoCStreams:       make([]*sctp.Stream, 0),
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

func decodeUint16FromBytes(buf []byte) uint16 {
	var ret uint16
	binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &ret)
	return ret
}

func (ols *OverlayServer) GetInfoForCtoCStream() (remotePeer mesh.PeerName, streamID uint16, err error) {
	util.OverlayDebugPrintln("before read remote PeerName from server stream.")
	var buf [8]byte
	// read PeerName of remote peer (this is internal protocol of gossip-overlay)
	_, err = ols.ServerStream.Read(buf[:])
	if err != nil {
		fmt.Println("err:", err)
		return math.MaxUint64, math.MaxUint16, err
	}
	decodedName := decodeUint64FromBytes(buf[:])
	remotePeerName := mesh.PeerName(decodedName)
	ols.gossipSession.RemoteAddress = &PeerAddress{remotePeerName}
	util.OverlayDebugPrintln("after read remote PeerName from server stream. buf:", buf)
	util.OverlayDebugPrintln("remotePeerName from stream: ", remotePeerName)

	util.OverlayDebugPrintln("before read stream ID to use from server stream")
	var buf2 [2]byte
	// read stream ID to use (this is internal protocol of gossip-overlay)
	_, err2 := ols.ServerStream.Read(buf2[:])
	if err2 != nil {
		fmt.Println("err:", err2)
		return math.MaxUint64, math.MaxUint16, err2
	}
	util.OverlayDebugPrintln("after read stream ID from server stream. buf2:", buf2)
	decodedStreamID := decodeUint16FromBytes(buf2[:])
	util.OverlayDebugPrintln("stream ID to use: ", decodedStreamID)

	return remotePeerName, decodedStreamID, nil
}

func (ols *OverlayServer) EstablishCtoCStream(remotePeer mesh.PeerName, streamID uint16) (*OverlayStream, error) {
	conn, err := ols.P.GossipDataMan.NewGossipSessionForClientToClient(remotePeer, streamID)
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("created a gossip session for client to client")

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err2 := sctp.Client(config)
	if err2 != nil {
		fmt.Println(err2)
		return nil, err2
	}

	stream, err3 := a.OpenStream(streamID, sctp.PayloadTypeWebRTCBinary)
	if err3 != nil {
		fmt.Println(err3)
		return nil, err3
	}
	util.OverlayDebugPrintln("opened a stream for client to client")

	stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	// TODO: write streamID to remotePeer through got Stream obj's Write as SYN
	//       and call Stream::Read to recv ACK (Read should block until ACK is received)

	overlayStream := NewOverlayStream(ols.P, stream, a, conn, streamID)

	return overlayStream, nil
}

// TODO: need to check second call of OverlayServer::Accept works collectly at view of ols.OriginalServerObj.AcceptStream call
func (ols *OverlayServer) Accept() (*OverlayStream, mesh.PeerName, error) {
	stream, err := ols.OriginalServerObj.AcceptStream()
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("accepted a stream")
	stream.SetReliabilityParams(true, sctp.ReliabilityTypeReliable, 0)
	ols.ServerStream = stream

	remotePeerName, streamID, err := ols.GetInfoForCtoCStream()
	if err != nil {
		util.OverlayDebugPrintln("err:", err)
		return nil, math.MaxUint64, err
	}

	ols.gossipSession.RemoteAddress = &PeerAddress{remotePeerName}

	// TODO: need to call Close of stream variable
	ols.ServerStream = nil

	overlayStream, err2 := ols.EstablishCtoCStream(remotePeerName, streamID)
	if err2 != nil {
		util.OverlayDebugPrintln("err2:", err2)
		return nil, math.MaxUint64, err2
	}

	util.OverlayDebugPrintln("end of OverlayServer.Accept")
	//return stream, remotePeerName, nil

	return overlayStream, remotePeerName, nil
}

func (ols *OverlayServer) InitInternalServerObj() error {
	conn, err := ols.P.GossipDataMan.NewGossipSessionForServer()
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("created a gossip session for server")

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err := sctp.Server(config)
	if err != nil {
		log.Panic(err)
	}
	util.OverlayDebugPrintln("created a server")

	ols.gossipSession = conn
	ols.OriginalServerObj = a

	return nil
}

func (ols *OverlayServer) Close() error {
	ols.gossipSession.Close()
	ols.gossipSession = nil
	ols.OriginalServerObj.Close()
	ols.OriginalServerObj = nil
	ols.StreamsMtx.Lock()
	for _, s := range ols.CtoCStreams {
		s.Close()
	}
	ols.CtoCStreams = nil
	ols.StreamsMtx.Unlock()

	return nil
}
