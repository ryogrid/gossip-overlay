package overlay

import (
	"fmt"
	"github.com/pion/datachannel"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/gossip"
	"github.com/ryogrid/gossip-overlay/overlay_setting"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math"
	"math/rand"
	"time"
)

var retryCntExceededErr = fmt.Errorf("retryCnt exceeded")

// wrapper of sctp.Client
type OverlayClient struct {
	peer            *gossip.GossipPeer
	remotePeerName  mesh.PeerName
	remotePeerHost  *string
	gossipMM        *gossip.GossipMessageManager
	HertbeatThFinCh *chan bool
}

func NewOverlayClient(p *gossip.GossipPeer, remotePeer mesh.PeerName, remotePeerHost string, gossipMM *gossip.GossipMessageManager) (*OverlayClient, error) {
	ret := &OverlayClient{
		peer:            p,
		remotePeerName:  remotePeer,
		remotePeerHost:  &remotePeerHost,
		gossipMM:        gossipMM,
		HertbeatThFinCh: nil,
	}

	return ret, nil
}

func genRandomStreamId() uint16 {
	dt := time.Now()
	unix := dt.UnixNano()
	randGen := rand.New(rand.NewSource(unix))
	return uint16(randGen.Uint32())
}

func (oc *OverlayClient) establishCtoCStreamInner(streamID uint16) (*sctp.Association, *gossip.GossipSession, error) {
	conn, err := oc.peer.GossipMM.NewGossipSessionForClientToClient(oc.remotePeerName, *oc.remotePeerHost, streamID)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}
	util.OverlayDebugPrintln("dialed gossip session for client to client", streamID, conn.StreamID)

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err2 := sctp.Client(config)
	if err2 != nil {
		fmt.Println(err2)
		return nil, nil, err2
	}

	return a, conn, nil
}

func (oc *OverlayClient) establishCtoCStreamClient(streamID uint16) (*OverlayStream, error) {
	a, gsess, _ := oc.establishCtoCStreamInner(streamID)

	util.OverlayDebugPrintln("opened a stream for client to client", streamID)

	loggerFactory := logging.NewDefaultLoggerFactory()

	cfg := &datachannel.Config{
		//ChannelType:          datachannel.ChannelTypePartialReliableRexmit,
		ChannelType: datachannel.ChannelTypeReliable,
		//ChannelType: datachannel.ChannelTypeReliableUnordered,
		//ReliabilityParameter: 5,
		//Label:         "data",
		Label:         "",
		LoggerFactory: loggerFactory,
		Negotiated:    false,
	}

	//dc, err := datachannel.Dial(a, 100, cfg)
	dc, err := datachannel.Dial(a, streamID, cfg)
	if err != nil {
		panic(err)
	}

	util.OverlayDebugPrintln("established a OverlayStream")

	//return dc, nil
	return NewOverlayStream(dc, oc, a, gsess), nil
}

func (oc *OverlayClient) establishCtoCStreamServer(streamID uint16) (*OverlayStream, error) {
	a, gsess, _ := oc.establishCtoCStreamInner(streamID)

	util.OverlayDebugPrintln("opened a stream for client to client", streamID)

	loggerFactory := logging.NewDefaultLoggerFactory()

	cfg := &datachannel.Config{
		//ChannelType:          datachannel.ChannelTypePartialReliableRexmit,
		ChannelType: datachannel.ChannelTypeReliable,
		//ChannelType: datachannel.ChannelTypeReliableUnordered,
		//ReliabilityParameter: 5,
		//Label:         "data",
		Label:         "",
		LoggerFactory: loggerFactory,
		Negotiated:    false,
	}

	//dc, err := datachannel.Dial(a, 100, cfg)
	dc, err := datachannel.Dial(a, streamID, cfg)
	//dc, err := datachannel.Accept(a, cfg)
	if err != nil {
		panic(err)
	}

	util.OverlayDebugPrintln("established a OverlayStream")

	//return dc, nil
	return NewOverlayStream(dc, oc, a, gsess), nil
}

func (oc *OverlayClient) NotifyOpenChReqToServer(streamId uint16) error {
	util.OverlayDebugPrintln("OverlayClient::NotifyOpenChReqToServer called", streamId)
	retryCnt := 0
retry:
	if retryCnt > 1 {
		fmt.Println(retryCntExceededErr)
		return retryCntExceededErr
	}
	// 2way
	err := oc.gossipMM.SendPingAndWaitPong(oc.remotePeerName, streamId, gossip.ServerSide, 5*time.Second, 0, []byte(oc.peer.GossipDataMan.Self.String()))
	if err != nil {
		// timeout
		util.OverlayDebugPrintln("GossipMessageManager.SendPingAndWaitPong: err:", err)
		retryCnt++
		goto retry
	}
	util.OverlayDebugPrintln("first GossipMessageManager.SendPingAndWaitPong call returned")
	//err = oc.gossipMM.SendPingAndWaitPong(oc.remotePeerName, streamId, gossip.ServerSide, 5*time.Second, 1, []byte(oc.peer.GossipDataMan.Self.String()))
	//if err != nil {
	//	// timeout
	//	util.OverlayDebugPrintln("GossipMessageManager.SendPingAndWaitPong: err:", err)
	//	retryCnt++
	//	goto retry
	//}
	//util.OverlayDebugPrintln("second GossipMessageManager.SendPingAndWaitPong call returned")
	return nil
}

func (oc *OverlayClient) OpenChannel(streamId uint16, isServer bool) (*OverlayStream, uint16, error) {
	streamId_ := uint16(0)
	if streamId != math.MaxUint16 {
		streamId_ = streamId
	} else {
		streamId_ = genRandomStreamId()
	}

	if streamId == math.MaxUint16 {
		// when client side initialization
		err := oc.NotifyOpenChReqToServer(streamId_)
		if err != nil {
			fmt.Println(err)
			return nil, math.MaxUint16, err
		}
	}

	var overlayStream *OverlayStream
	var err error
	if isServer {
		overlayStream, err = oc.establishCtoCStreamServer(streamId_)
		if err != nil {
			panic(err)
		}
	} else {
		overlayStream, err = oc.establishCtoCStreamClient(streamId_)
		if err != nil {
			panic(err)
		}
	}

	if err != nil {
		panic(err)
		//fmt.Println(err)
		//return nil, math.MaxUint16, err
	}

	util.OverlayDebugPrintln("end of OverlayClient::OpenChannel")

	// start heartbeat thread
	//tmpCh := make(chan bool)
	//oc.HertbeatThFinCh = &tmpCh
	//go oc.heaertbeatSendingTh(overlayStream.gsess)

	return overlayStream, streamId_, nil
}

func (oc *OverlayClient) heaertbeatSendingTh(gsess *gossip.GossipSession) {
	t := time.NewTicker(overlay_setting.HEARTBEAT_INTERVAL * time.Second) // interval
loop:
	for {
		select {
		case <-t.C: // interval reached
			// send heartbeat packet
			failCnt := 0
			err := oc.gossipMM.SendPingAndWaitPong(oc.remotePeerName, 0, gossip.ClientSide, 30*time.Second, math.MaxUint32, []byte{})
			if err != nil {
				failCnt++
				util.OverlayDebugPrintln("call GossipMessageManager::SendPingAndWaitPong on heartbeatSendingTh: failCnt is incremented to ", failCnt)
			}
			err = oc.gossipMM.SendPingAndWaitPong(oc.remotePeerName, 0, gossip.ClientSide, 30*time.Second, math.MaxUint32, []byte{})
			if err != nil {
				failCnt++
				util.OverlayDebugPrintln("call GossipMessageManager::SendPingAndWaitPong on heartbeatSendingTh: failCnt is incremented to ", failCnt)
			}
			util.OverlayDebugPrintln("call GossipMessageManager::SendPingAndWaitPong on heartbeatSendingTh: failCnt is ", failCnt)
			if failCnt == 2 {
				// connection lost
				// end heartbeat thread
				// and deactivate gossip session
				gsess.IsActive = false
				break loop
			} else {
				// connection alive
				// and do nothing
			}
		case <-*oc.HertbeatThFinCh:
			break loop
		}
	}
	t.Stop()
}

func (oc *OverlayClient) Destroy() error {
	//close(*oc.HertbeatThFinCh)
	return nil
}
