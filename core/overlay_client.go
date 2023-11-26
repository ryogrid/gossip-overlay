package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math"
	"math/rand"
	"time"
)

// wrapper of sctp.Client
type OverlayClient struct {
	P                             *Peer
	OriginalClientObj             *sctp.Association
	StreamToNotifySelfInfo        *sctp.Stream
	GossipSessionToNotifySelfInfo *GossipSession
	RemotePeerName                mesh.PeerName
}

// TODO: need to implement wrapper class of sctp.Stream for close related resouces properly (client side)

func NewOverlayClient(p *Peer, remotePeer mesh.PeerName) (*OverlayClient, error) {
	ret := &OverlayClient{
		P:                             p,
		OriginalClientObj:             nil,
		StreamToNotifySelfInfo:        nil,
		GossipSessionToNotifySelfInfo: nil,
		RemotePeerName:                remotePeer,
	}

	err := ret.PrepareNewClientObj()
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

func encodeUint16ToBytes(streamId uint16) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, &streamId)
	return buf.Bytes()
}

func (oc *OverlayClient) PrepareNewClientObj() error {
	conn, err := oc.P.GossipDataMan.NewGossipSessionForClientToServer(oc.RemotePeerName)
	if err != nil {
		fmt.Println(err)
		return err
	}
	oc.GossipSessionToNotifySelfInfo = conn
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

	util.OverlayDebugPrintln("created a client")

	return nil
}

func genRandomStreamId() uint16 {
	dt := time.Now()
	unix := dt.UnixNano()
	randGen := rand.New(rand.NewSource(unix))
	return uint16(randGen.Uint32())
}

func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*sctp.Stream, error) {
	// TODO: need to create Association obj with sctp.Client method
	// TODO: need to call Association::OpenStream to remotePeer with streamID then get Stream obj
	// TODO: read streamID from remotePeer through got Stream obj's Read as SYN (Read should block until ACK is received)
	//       and call Stream::Write to send ACK

	return stream, nil
}

func (oc *OverlayClient) innerOpenStreamToServer() (streamID uint16, err error) {
	stream, err := oc.OriginalClientObj.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		fmt.Println(err)
		return math.MaxUint16, err
	}
	oc.StreamToNotifySelfInfo = stream
	util.OverlayDebugPrintln("opened a stream")

	stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	util.OverlayDebugPrintln("before write self PeerName to stream")
	// write my PeerName (this is internal protocol of gossip-overlay)
	sendData := encodeUint64ToBytes(uint64(oc.P.GossipDataMan.Self))
	_, err = stream.Write(sendData)
	if err != nil {
		fmt.Println(err)
		return math.MaxUint16, err
	}
	util.OverlayDebugPrintln("after write self PeerName to stream")

	util.OverlayDebugPrintln("before write stream ID to use to stream")
	// write my PeerName (this is internal protocol of gossip-overlay)
	streamId := genRandomStreamId()
	sendData2 := encodeUint16ToBytes(streamId)
	_, err = stream.Write(sendData2)
	if err != nil {
		fmt.Println(err)
		return math.MaxUint16, err
	}
	util.OverlayDebugPrintln("after write stream ID to use to stream")

	return streamId, nil
}

func (oc *OverlayClient) OpenStream() (*sctp.Stream, error) {
	streamIdToUse, err := oc.innerOpenStreamToServer()
	if err != nil {
		util.OverlayDebugPrintln("err:", err)
		return nil, err
	}

	var buf [1]byte
	// wait until server side stream close
	_, err2 := oc.StreamToNotifySelfInfo.Read(buf[:])
	if err2 != nil {
		util.OverlayDebugPrintln("err2:", err2)
		util.OverlayDebugPrintln("may be stream is closed by server side")
	}

	// TODO: need to clear used local buffer for sending self info (OverlayClient::OpenStream)

	stream, err3 := oc.establishCtoCStream(streamIdToUse)

	// TODO: need to wrapp stream obj with needed obj reference (OverlayClient::OpenStream)

	return overlayStream, nil
}

func (oc *OverlayClient) Close() error {
	// TODO: need implement OverlayClient::Close

	err := oc.GossipSessionToNotifySelfInfo.Close()
	if err != nil {
		fmt.Println(err)
	}
	err = oc.OriginalClientObj.Close()
	if err != nil {
		fmt.Println(err)
	}
	err = oc.StreamToNotifySelfInfo.Close()
	if err != nil {
		fmt.Println(err)
	}
	return err
}
