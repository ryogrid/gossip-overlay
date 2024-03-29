package overlay

import (
	"fmt"
	"github.com/ryogrid/gossip-overlay/gossip"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math"
)

type clientInfo struct {
	remotePeerName mesh.PeerName
	remotePeerHost string
	streamID       uint16
}

// wrapper of sctp.Server
type OverlayServer struct {
	peer                 *gossip.GossipPeer
	info4OLChannelRecvCh chan *clientInfo
	gossipMM             *gossip.GossipMessageManager
}

func NewOverlayServer(p *gossip.GossipPeer, gossipMM *gossip.GossipMessageManager) (*OverlayServer, error) {
	ret := &OverlayServer{
		peer:                 p,
		info4OLChannelRecvCh: nil,
		gossipMM:             gossipMM,
	}

	// start root handling thread for client info notify packet
	ret.InitClientInfoNotifyPktRootHandlerTh()

	return ret, nil
}

/*
// thread for handling client info notify packet of each client
func (ols *OverlayServer) newHandshakeHandlingThServSide(remotePeer mesh.PeerName, streamID uint16,
	recvPktCh <-chan *gossip.GossipPacket, finNotifyCh chan<- *clientInfo, notifyErrCh chan<- *clientInfo) {
	pkt := <-recvPktCh
	if pkt.SeqNum != 0 {
		// exit thread as handshake failed
		notifyErrCh <- &clientInfo{remotePeer, pkt.FromPeerHost, streamID}
		return
	}
	util.OverlayDebugPrintln(pkt)
	ols.gossipMM.SendPongPktToClient(remotePeer, streamID, 0)

	done := make(chan interface{})

	// 30 seconds timeout
	go func() {
		time.Sleep(30 * time.Second)
		close(done)
	}()

loop:
	for {
		select {
		case pkt = <-recvPktCh:
			util.OverlayDebugPrintln(pkt)
			if pkt.SeqNum == 1 {
				util.OverlayDebugPrintln("overlayServer::newHandshakeHandlingThServSide: received expected packet")
				// second packat
				break loop
			} else {
				util.OverlayDebugPrintln("overlayServer::newHandshakeHandlingThServSide: received unexpected packet")
				// exit thread as handshake failed
				notifyErrCh <- &clientInfo{remotePeer, pkt.FromPeerHost, streamID}
				return
			}
		case <-done:
			// timeout reached
			// exit thread as handshake failed
			notifyErrCh <- &clientInfo{remotePeer, pkt.FromPeerHost, streamID}
			return
		}
	}
	ols.gossipMM.SendPongPktToClient(remotePeer, streamID, 1)

	finNotifyCh <- &clientInfo{remotePeer, pkt.FromPeerHost, streamID}
}
*/

func (ols *OverlayServer) ClientInfoNotifyPktRootHandlerTh() {
	util.OverlayDebugPrintln("overlayServer::ClientInfoNotifyPktRootHandlerTh: start")

	// "peer name"-"stream id" -> channel to appropriate packet handling thread
	//handshakePktHandleThChans := sync.Map{}
	//finCh := make(chan *clientInfo, 1)
	//notifyErrCh := make(chan *clientInfo, 1)

	for {
		util.OverlayDebugPrintln("overlayServer::ClientInfoNotifyPktRootHandlerTh: waiting pkt")
		select {
		case pkt := <-ols.gossipMM.NotifyPktChForServerSide:
			util.OverlayDebugPrintln("overlayServer::ClientInfoNotifyPktRootHandlerTh: received pkt:", pkt)
			// pass packet to appropriate handling thread (if not exist, spawn new handling thread)
			//key := pkt.FromPeer.String() + "-" + string(pkt.StreamID)
			//if ch, ok := handshakePktHandleThChans.Load(key); ok {
			//	ch.(chan *gossip.GossipPacket) <- pkt
			//} else {
			//	newCh := make(chan *gossip.GossipPacket, 1)
			//
			//	handshakePktHandleThChans.Store(key, newCh)
			//	go ols.newHandshakeHandlingThServSide(pkt.FromPeer, pkt.StreamID, newCh, finCh, notifyErrCh)
			//	newCh <- pkt
			//}
			ols.gossipMM.SendPongPktToClient(pkt.FromPeer, pkt.StreamID, 0)
			ols.info4OLChannelRecvCh <- &clientInfo{pkt.FromPeer, pkt.FromPeerHost, pkt.StreamID}
			//case finInfo := <-finCh:
			//	// remove channel to notify origin thread
			//	handshakePktHandleThChans.Delete(finInfo.remotePeerName.String() + "-" + string(finInfo.streamID))
			//	// notify new request info to Accept method
			//	ols.info4OLChannelRecvCh <- finInfo
			//case errRemote := <-notifyErrCh:
			//	// handshake failed
			//	handshakePktHandleThChans.Delete(errRemote.remotePeerName.String() + "-" + string(errRemote.streamID))
		}
	}
	util.OverlayDebugPrintln("overlayServer::ClientInfoNotifyPktRootHandlerTh: end")
}

func (ols *OverlayServer) InitClientInfoNotifyPktRootHandlerTh() error {
	ols.info4OLChannelRecvCh = make(chan *clientInfo, 1)

	go ols.ClientInfoNotifyPktRootHandlerTh()

	return nil
}

func (ols *OverlayServer) Accept() (*OverlayStream, mesh.PeerName, *string, uint16, error) {
	clInfo, more := <-ols.info4OLChannelRecvCh
	if !more {
		return nil, math.MaxUint64, nil, math.MaxUint16, fmt.Errorf("OverlayServer is closed")
	}
	//fmt.Println(clInfo)
	olc, err := NewOverlayClient(ols.peer, clInfo.remotePeerName, clInfo.remotePeerHost, ols.gossipMM)
	if err != nil {
		panic(err)
	}

	os, _, err2 := olc.OpenChannel(clInfo.streamID)
	if err2 != nil {
		return nil, math.MaxUint64, nil, clInfo.streamID, err2
	}

	return os, clInfo.remotePeerName, &clInfo.remotePeerHost, clInfo.streamID, nil
}

func (ols *OverlayServer) Close() error {
	// TODO: need to implement overlayServer::Close
	close(ols.info4OLChannelRecvCh)

	return nil
}
