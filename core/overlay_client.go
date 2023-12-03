package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/pion/datachannel"
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

func (oc *OverlayClient) establishCtoCStreamInner(remotePeerName mesh.PeerName, streamID uint16) (*sctp.Association, error) {
	conn, err := oc.P.GossipDataMan.NewGossipSessionForClientToClient(remotePeerName, streamID)
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

	//stream, err3 := a.OpenStream(streamID, sctp.PayloadTypeWebRTCBinary)
	//if err3 != nil {
	//	fmt.Println(err3)
	//	return nil, err3
	//}

	return a, nil
}

// func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*OverlayStream, error) {
func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*datachannel.DataChannel, *datachannel.DataChannel, error) {
	var a1 *sctp.Association = nil
	var a2_1 *sctp.Association = nil
	var a2_2 *sctp.Association = nil
	var a3 *sctp.Association = nil

	if oc.P.GossipDataMan.Self == 1 { // dial side (to peer-2)
		a1, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 1)
	} else if oc.P.GossipDataMan.Self == 2 { // accept side x 2 (from peer1 and peer3)
		a2_1, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 1)
		a2_2, _ = oc.establishCtoCStreamInner(3, 2)
	} else if oc.P.GossipDataMan.Self == 3 { // dial side (to peer-2)
		a3, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 2)
	} else {
		panic("invalid destname")
	}

	util.OverlayDebugPrintln("opened a stream for client to client", streamID)
	//stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	//sendSYN := func() error {
	//	// send SYN
	//	sendData := encodeUint16ToBytes(streamID)
	//	_, err4_ := stream.Write(sendData)
	//	if err4_ != nil {
	//		fmt.Println(err4_)
	//		//return nil, err4
	//		return err4_
	//	}
	//	return nil
	//}
	//
	//waitSYN := func() error {
	//	// wait until SYN is received
	//	buf := make([]byte, 2)
	//	n, err4 := stream.Read(buf)
	//	recvedStreamID := decodeUint16FromBytes(buf)
	//	if err4 != nil || n != 2 || recvedStreamID != streamID {
	//		fmt.Println("err:", err4, " n:", n, " recvedStreamID:", recvedStreamID)
	//		//return nil, err4
	//		return err4
	//	}
	//	return nil
	//}
	//
	//sendACK := func() error {
	//	// send ACK
	//	sendData := encodeUint16ToBytes(streamID)
	//	_, err5 := stream.Write(sendData)
	//	if err5 != nil {
	//		fmt.Println(err5)
	//		//return nil, err5
	//		return err5
	//	}
	//	return nil
	//}
	//
	//waitACK := func() error {
	//	// wait until ACK is received
	//	buf := make([]byte, 2)
	//	n, err5_ := stream.Read(buf)
	//	recvedStreamID := decodeUint16FromBytes(buf)
	//	if err5_ != nil || n != 2 || recvedStreamID != streamID {
	//		fmt.Println("err:", err5_, " n:", n, " recvedStreamID:", recvedStreamID)
	//		//return nil, err5
	//		return err5_
	//	}
	//	return nil
	//}

	loggerFactory := logging.NewDefaultLoggerFactory()
	var dc1 *datachannel.DataChannel = nil
	var dc2 *datachannel.DataChannel = nil
	var err error = nil

	// TODO: temporal impl
	if oc.P.GossipDataMan.Self == 1 { // dial side (to peer-2)
		cfg := &datachannel.Config{
			ChannelType:          datachannel.ChannelTypePartialReliableRexmit,
			ReliabilityParameter: 0,
			Label:                "data",
			LoggerFactory:        loggerFactory,
		}

		dc1, err = datachannel.Dial(a1, 100, cfg)
		if err != nil {
			panic(err)
		}
	} else if oc.P.GossipDataMan.Self == 2 { // accept side x 2 (from peer1 and peer3)
		dc1, err = datachannel.Accept(a2_1, &datachannel.Config{
			LoggerFactory: loggerFactory,
		})
		if err != nil {
			panic(err)
		}

		dc2, err = datachannel.Accept(a2_2, &datachannel.Config{
			LoggerFactory: loggerFactory,
		})
		if err != nil {
			panic(err)
		}
	} else if oc.P.GossipDataMan.Self == 3 { // dial side (to peer-2)
		cfg := &datachannel.Config{
			ChannelType:          datachannel.ChannelTypePartialReliableRexmit,
			ReliabilityParameter: 0,
			Label:                "data",
			LoggerFactory:        loggerFactory,
		}

		dc1, err = datachannel.Dial(a3, 101, cfg)
		if err != nil {
			panic(err)
		}
	} else {
		panic("invalid destname")
	}

	//overlayStream := NewOverlayStream(oc.P, stream, a, conn, streamID)
	util.OverlayDebugPrintln("established a OverlayStream")

	//return overlayStream, nil
	return dc1, dc2, nil
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

// func (oc *OverlayClient) OpenStream(streamId uint16) (*OverlayStream, error) {
func (oc *OverlayClient) OpenStream(streamId uint16) (*datachannel.DataChannel, *datachannel.DataChannel, error) {
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
	overlayStream1, overlayStream2, err3 := oc.establishCtoCStream(streamId)
	if err3 != nil {
		fmt.Println(err3)
		return nil, nil, err3
	}

	util.OverlayDebugPrintln("end of OverlayClient::OpenStream")

	return overlayStream1, overlayStream2, nil
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
