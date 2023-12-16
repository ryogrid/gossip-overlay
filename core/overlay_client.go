package core

import (
	"bytes"
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
	P              *Peer
	RemotePeerName mesh.PeerName
}

func NewOverlayClient(p *Peer, remotePeer mesh.PeerName) (*OverlayClient, error) {
	ret := &OverlayClient{
		P:              p,
		RemotePeerName: remotePeer,
	}

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

	return a, nil
}

func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*datachannel.DataChannel, *datachannel.DataChannel, *sctp.Association, error) {
	var a1 *sctp.Association = nil
	var a2_1 *sctp.Association = nil
	var a2_2 *sctp.Association = nil
	var a3_1 *sctp.Association = nil
	var a3_2 *sctp.Association = nil

	if oc.P.GossipDataMan.Self == 1 { // dial side (to peer-2)
		a1, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 1)
	} else if oc.P.GossipDataMan.Self == 2 { // accept side x 2 (from peer1 and peer3)
		a2_1, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 1)
		a2_2, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 2)
	} else if oc.P.GossipDataMan.Self == 3 { // dial side (to peer-2)
		a3_1, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 1)
		a3_2, _ = oc.establishCtoCStreamInner(oc.RemotePeerName, 2)
	} else {
		panic("invalid destname")
	}

	util.OverlayDebugPrintln("opened a stream for client to client", streamID)

	loggerFactory := logging.NewDefaultLoggerFactory()
	var dc1 *datachannel.DataChannel = nil
	var dc2 *datachannel.DataChannel = nil
	var err error = nil

	// TODO: temporal impl (OverlayClient::establishCtoCStream)
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
	} else if oc.P.GossipDataMan.Self == 2 { // accept side and dial side (from peer3 and to peer3)
		dc1, err = datachannel.Accept(a2_1, &datachannel.Config{
			LoggerFactory: loggerFactory,
		})
		if err != nil {
			panic(err)
		}

		cfg := &datachannel.Config{
			ChannelType:          datachannel.ChannelTypeReliable,
			ReliabilityParameter: 0,
			Label:                "data",
			LoggerFactory:        loggerFactory,
		}

		dc2, err = datachannel.Dial(a2_2, 101, cfg)
		if err != nil {
			panic(err)
		}
	} else if oc.P.GossipDataMan.Self == 3 { // dial side (to peer-2)
		cfg := &datachannel.Config{
			ChannelType:          datachannel.ChannelTypeReliable,
			ReliabilityParameter: 0,
			Label:                "data",
			LoggerFactory:        loggerFactory,
		}

		dc1, err = datachannel.Dial(a3_1, 100, cfg)
		if err != nil {
			panic(err)
		}

		dc2, err = datachannel.Accept(a3_2, &datachannel.Config{
			LoggerFactory: loggerFactory,
		})
		if err != nil {
			panic(err)
		}
	} else {
		panic("invalid destname")
	}

	util.OverlayDebugPrintln("established a OverlayStream")

	// TODO: need to modify to return one datachannel (OverlayClient::establishCtoCStream)
	return dc1, dc2, a2_2, nil
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

func (oc *OverlayClient) NotifyOpenChReqToServer(id uint16) {
	// TODO: need to implement (OverlayClient::NotifyOpenChReqToServer)

	panic("not implemented")
}

// func (oc *OverlayClient) OpenStream(streamId uint16) (*OverlayStream, error) {
func (oc *OverlayClient) OpenStream(streamId uint16) (*datachannel.DataChannel, *datachannel.DataChannel, *sctp.Association, error) {
	oc.NotifyOpenChReqToServer(streamId)

	overlayStream1, overlayStream2, a2_2, err3 := oc.establishCtoCStream(streamId)
	if err3 != nil {
		fmt.Println(err3)
		return nil, nil, nil, err3
	}

	util.OverlayDebugPrintln("end of OverlayClient::OpenStream")

	return overlayStream1, overlayStream2, a2_2, nil
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

	return nil
}
