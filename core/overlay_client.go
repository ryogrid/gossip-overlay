package core

import (
	"bytes"
	"context"
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
	P                             *Peer
	OriginalClientObj             *sctp.Association
	StreamToNotifySelfInfo        *sctp.Stream
	GossipSessionToNotifySelfInfo *GossipSession
	RemotePeerName                mesh.PeerName
}

func NewOverlayClient(p *Peer, remotePeer mesh.PeerName) (*OverlayClient, error) {
	ret := &OverlayClient{
		P:                             p,
		OriginalClientObj:             nil,
		StreamToNotifySelfInfo:        nil,
		GossipSessionToNotifySelfInfo: nil,
		RemotePeerName:                remotePeer,
	}

	//err := ret.PrepareNewClientObj()
	//if err != nil {
	//	fmt.Println(err)
	//	return nil, err
	//}

	//return ret, err
	return ret, nil
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

//func (oc *OverlayClient) PrepareNewClientObj() error {
//	conn, err := oc.P.GossipDataMan.NewGossipSessionForClientToServer(oc.RemotePeerName)
//	if err != nil {
//		fmt.Println(err)
//		return err
//	}
//	oc.GossipSessionToNotifySelfInfo = conn
//	util.OverlayDebugPrintln("dialed gossip session to server")
//
//	config := sctp.Config{
//		NetConn:       conn,
//		LoggerFactory: logging.NewDefaultLoggerFactory(),
//	}
//	a, err2 := sctp.Client(config)
//	if err2 != nil {
//		fmt.Println(err2)
//		return err2
//	}
//
//	oc.OriginalClientObj = a
//
//	util.OverlayDebugPrintln("prepared a inner client")
//
//	return nil
//}

func genRandomStreamId() uint16 {
	dt := time.Now()
	unix := dt.UnixNano()
	randGen := rand.New(rand.NewSource(unix))
	return uint16(randGen.Uint32())
}

func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*OverlayStream, error) {
	conn, err := oc.P.GossipDataMan.NewGossipSessionForClientToClient(oc.RemotePeerName, streamID)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	util.OverlayDebugPrintln("dialed gossip session for client to client", streamID, conn.StreamID)

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
	util.OverlayDebugPrintln("opened a stream for client to client", streamID)

	stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	// wait until SYN is received
	buf := make([]byte, 2)
	n, err4 := stream.Read(buf)
	recvedStreamID := decodeUint16FromBytes(buf)
	if err4 != nil || n != 2 || recvedStreamID != streamID {
		fmt.Println("err:", err4, " n:", n, " recvedStreamID:", recvedStreamID)
		return nil, err4
	}

	// send ACK
	sendData := encodeUint16ToBytes(streamID)
	_, err5 := stream.Write(sendData)
	if err5 != nil {
		fmt.Println(err5)
		return nil, err5
	}

	overlayStream := NewOverlayStream(oc.P, stream, a, conn, streamID)

	return overlayStream, nil
}

//func (oc *OverlayClient) innerOpenStreamToServer() (streamID uint16, err error) {
//	stream, err := oc.OriginalClientObj.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
//	if err != nil {
//		fmt.Println(err)
//		return math.MaxUint16, err
//	}
//	oc.StreamToNotifySelfInfo = stream
//	util.OverlayDebugPrintln("opened a stream to server")
//
//	stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)
//
//	util.OverlayDebugPrintln("before write self PeerName to stream")
//	// write my PeerName (this is internal protocol of gossip-overlay)
//	sendData := encodeUint64ToBytes(uint64(oc.P.GossipDataMan.Self))
//	_, err2 := stream.Write(sendData)
//	if err2 != nil {
//		fmt.Println(err2)
//		return math.MaxUint16, err2
//	}
//	util.OverlayDebugPrintln("after write self PeerName to stream")
//
//	util.OverlayDebugPrintln("before write stream ID to use to stream")
//	// write my PeerName (this is internal protocol of gossip-overlay)
//	streamId := genRandomStreamId()
//	sendData2 := encodeUint16ToBytes(streamId)
//	_, err3 := stream.Write(sendData2)
//	if err3 != nil {
//		fmt.Println(err3)
//		return math.MaxUint16, err3
//	}
//	util.OverlayDebugPrintln("after write stream ID to use to stream")
//
//	return streamId, nil
//}

func (oc *OverlayClient) OpenStream(streamId uint16) (*OverlayStream, error) {
	//streamIdToUse, err := oc.innerOpenStreamToServer()
	//if err != nil {
	//	util.OverlayDebugPrintln("err:", err)
	//	return nil, err
	//}
	//
	//util.OverlayDebugPrintln("before waiting for server side stream close")
	//
	//var buf [1]byte
	//// wait until server side stream close
	//_, err2 := oc.StreamToNotifySelfInfo.Read(buf[:])
	//if err2 != nil {
	//	util.OverlayDebugPrintln("err2:", err2)
	//	util.OverlayDebugPrintln("may be stream is closed by server side")
	//}
	//
	//util.OverlayDebugPrintln("after waiting for server side stream close")
	//
	////// resouce releases: stream to notify self info and related resources
	////oc.Close()
	//oc.GossipSessionToNotifySelfInfo = nil
	//oc.OriginalClientObj = nil
	//oc.StreamToNotifySelfInfo = nil

	//overlayStream, err3 := oc.establishCtoCStream(streamIdToUse)
	overlayStream, err3 := oc.establishCtoCStream(streamId)
	if err3 != nil {
		fmt.Println(err3)
		return nil, err3
	}

	util.OverlayDebugPrintln("end of OverlayClient::OpenStream")

	return overlayStream, nil
}

func (oc *OverlayClient) Close() error {
	//err := oc.GossipSessionToNotifySelfInfo.Close()
	//if err != nil {
	//	fmt.Println(err)
	//}
	//err := oc.OriginalClientObj.Close()
	//if err != nil {
	//	fmt.Println(err)
	//}
	err2 := oc.StreamToNotifySelfInfo.Close()
	if err2 != nil {
		fmt.Println(err2)
	}
	err3 := oc.OriginalClientObj.Shutdown(context.Background())
	if err3 != nil {
		fmt.Println(err3)
	}

	return nil
}
