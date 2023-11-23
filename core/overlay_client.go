package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math/rand"
	"time"
)

// wrapper of sctp.Client
type OverlayClient struct {
	P                 *Peer
	OriginalClientObj *sctp.Association
	Stream            *sctp.Stream
	GossipSession     *GossipSession
}

func NewOverlayClient(p *Peer, remotePeer mesh.PeerName) (*OverlayClient, error) {
	ret := &OverlayClient{
		P:                 p,
		OriginalClientObj: nil,
		Stream:            nil,
		GossipSession:     nil,
	}

	err := ret.PrepareNewClientObj(remotePeer)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return ret, err
}

func encodeUint64ToBytes(peerName uint64) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, &peerName)
	return buf.Bytes()
}

func (oc *OverlayClient) PrepareNewClientObj(remotePeer mesh.PeerName) error {
	conn, err := oc.P.GossipDataMan.NewGossipSessionForClient(remotePeer)
	if err != nil {
		fmt.Println(err)
		return err
	}
	oc.GossipSession = conn
	//defer func() {
	//	if closeErr := conn.Close(); closeErr != nil {
	//		panic(err)
	//	}
	//}()
	util.OverlayDebugPrintln("dialed gossip session")

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err2 := sctp.Client(config)
	if err2 != nil {
		fmt.Println(err2)
		return err2
	}

	oc.OriginalClientObj = a

	//defer func() {
	//	if closeErr := a.Close(); closeErr != nil {
	//		panic(err)
	//	}
	//}()
	util.OverlayDebugPrintln("created a client")

	return nil
}

func (oc *OverlayClient) OpenStream() (*sctp.Stream, error) {
	//stream, err := a.OpenStream(0, sctp.PayloadTypeWebRTCString)
	dt := time.Now()
	unix := dt.UnixNano()
	randGen := rand.New(rand.NewSource(unix))
	stream, err := oc.OriginalClientObj.OpenStream(uint16(randGen.Uint32()), sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	oc.Stream = stream
	//defer func() {
	//	if closeErr := stream.Close(); closeErr != nil {
	//		panic(err)
	//	}
	//}()
	util.OverlayDebugPrintln("opened a stream")

	// set unordered = true and 10ms treshold for dropping packets
	//stream.SetReliabilityParams(true, sctp.ReliabilityTypeTimed, 10)
	stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	// write my PeerName (this is internal protocol of gossip-overlay)
	sendData := encodeUint64ToBytes(uint64(oc.P.GossipDataMan.Self))
	_, err = stream.Write(sendData)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return stream, nil
}

func (oc *OverlayClient) Close() error {
	err := oc.GossipSession.Close()
	if err != nil {
		fmt.Println(err)
	}
	err = oc.OriginalClientObj.Close()
	if err != nil {
		fmt.Println(err)
	}
	err = oc.Stream.Close()
	if err != nil {
		fmt.Println(err)
	}
	return err
}
