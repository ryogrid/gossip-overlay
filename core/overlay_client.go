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

func (oc *OverlayClient) establishCtoCStream(streamID uint16) (*datachannel.DataChannel, error) {
	a, _ := oc.establishCtoCStreamInner(streamID)

	util.OverlayDebugPrintln("opened a stream for client to client", streamID)

	loggerFactory := logging.NewDefaultLoggerFactory()

	cfg := &datachannel.Config{
		ChannelType:          datachannel.ChannelTypePartialReliableRexmit,
		ReliabilityParameter: 0,
		Label:                "data",
		LoggerFactory:        loggerFactory,
	}

	dc, err := datachannel.Dial(a, 100, cfg)
	if err != nil {
		panic(err)
	}

	util.OverlayDebugPrintln("established a OverlayStream")

	return dc, nil
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

func (oc *OverlayClient) OpenChannel() (*datachannel.DataChannel, uint16, error) {
	streamId := genRandomStreamId()
	oc.NotifyOpenChReqToServer(streamId)

	overlayStream, err := oc.establishCtoCStream(streamId)
	if err != nil {
		panic(err)
		//fmt.Println(err)
		//return nil, math.MaxUint16, err
	}

	util.OverlayDebugPrintln("end of OverlayClient::OpenChannel")

	return overlayStream, streamId, nil
}

func (oc *OverlayClient) Destroy() error {
	return nil
}
