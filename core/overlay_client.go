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
	GossipMM       *GossipMessageManager
}

func NewOverlayClient(p *Peer, remotePeer mesh.PeerName, gossipMM *GossipMessageManager) (*OverlayClient, error) {
	ret := &OverlayClient{
		P:              p,
		RemotePeerName: remotePeer,
		GossipMM:       gossipMM,
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

func (oc *OverlayClient) establishCtoCStreamInner(streamID uint16) (*sctp.Association, error) {
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

	return a, nil
}

func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*datachannel.DataChannel, *datachannel.DataChannel, *sctp.Association, error) {
	// TODO: need to modifiy implementation to be general purpose (OverlayClient::establishCtoCStream)
	//       and return value should be changed appropriately

	var a1 *sctp.Association = nil
	var a2_1 *sctp.Association = nil
	var a2_2 *sctp.Association = nil
	var a3_1 *sctp.Association = nil
	var a3_2 *sctp.Association = nil

	if oc.P.GossipDataMan.Self == 1 { // dial side (to peer-2)
		a1, _ = oc.establishCtoCStreamInner(1)
	} else if oc.P.GossipDataMan.Self == 2 { // accept side x 2 (from peer1 and peer3)
		a2_1, _ = oc.establishCtoCStreamInner(1)
		a2_2, _ = oc.establishCtoCStreamInner(2)
	} else if oc.P.GossipDataMan.Self == 3 { // dial side (to peer-2)
		a3_1, _ = oc.establishCtoCStreamInner(1)
		a3_2, _ = oc.establishCtoCStreamInner(2)
	} else {
		panic("invalid destname")
	}

	util.OverlayDebugPrintln("opened a stream for client to client", streamID)

	loggerFactory := logging.NewDefaultLoggerFactory()
	var dc1 *datachannel.DataChannel = nil
	var dc2 *datachannel.DataChannel = nil
	var err error = nil

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

//// thread for handling handshake packet
//func (ols *OverlayClient) newHandshakeHandlingThClientSide(remotePeer mesh.PeerName, streamID uint16,
//	recvPktCh chan *GossipPacket, finNotifyCh chan *ClientInfo) {
//	// TODO: need to initialization of variables for handshake state recognition (OverlayServer::newHandshakeHandlingThClientSide)
//	// TODO: need to send ping packet (OverlayServer::newHandshakeHandlingThClientSide)
//
//	pkt := <-recvPktCh
//	fmt.Println(pkt)
//	// TODO: need to implement (OverlayServer::newHandshakeHandlingTh)
//	//       - send and recv ping-pong packet
//
//	finNotifyCh <- &ClientInfo{remotePeer, streamID}
//}

func (oc *OverlayClient) NotifyOpenChReqToServer(streamId uint16) {
retry:
	// 4way
	err := oc.GossipMM.SendPingAndWaitPong(oc.RemotePeerName, streamId, ClientSide, []byte(oc.P.GossipDataMan.Self.String()))
	if err != nil {
		// timeout
		util.OverlayDebugPrintln("GossipMessageManager.SendPingAndWaitPong: err:", err)
		goto retry
	}
	oc.GossipMM.SendPingAndWaitPong(oc.RemotePeerName, streamId, ClientSide, []byte(oc.P.GossipDataMan.Self.String()))
	if err != nil {
		// timeout
		util.OverlayDebugPrintln("GossipMessageManager.SendPingAndWaitPong: err:", err)
		goto retry
	}
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
	//err2 := oc.StreamToNotifySelfInfo.Close()
	//if err2 != nil {
	//	fmt.Println(err2)
	//}

	return nil
}
