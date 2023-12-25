package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pion/datachannel"
	"github.com/ryogrid/gossip-overlay/gossip"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"sync"
	"time"
)

type ClientInfo struct {
	RemotePeerName mesh.PeerName
	StreamID       uint16
}

// wrapper of sctp.Server
type OverlayServer struct {
	P                    *Peer
	Info4OLChannelRecvCh chan *ClientInfo
	GossipMM             *gossip.GossipMessageManager
}

func NewOverlayServer(p *Peer, gossipMM *gossip.GossipMessageManager) (*OverlayServer, error) {
	ret := &OverlayServer{
		P:                    p,
		Info4OLChannelRecvCh: nil,
		GossipMM:             gossipMM,
	}

	// start root handling thread for client info notify packet
	ret.InitClientInfoNotifyPktRootHandlerTh()

	return ret, nil
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

func (ols *OverlayServer) sendPongPktToClient(remotePeer mesh.PeerName, streamID uint16, seqNum uint64) error {
	err := ols.GossipMM.SendToRemote(remotePeer, streamID, gossip.ClientSide, seqNum, []byte{})
	if err != nil {
		//fmt.Println(err)
		//return err
		panic(err)
	}
	return nil
}

// thread for handling client info notify packet of each client
func (ols *OverlayServer) newHandshakeHandlingThServSide(remotePeer mesh.PeerName, streamID uint16,
	recvPktCh <-chan *gossip.GossipPacket, finNotifyCh chan<- *ClientInfo, notifyErrCh chan<- *ClientInfo) {
	pkt := <-recvPktCh
	if pkt.SeqNum != 0 {
		// exit thread as handshake failed
		notifyErrCh <- &ClientInfo{remotePeer, streamID}
		return
	}
	util.OverlayDebugPrintln(pkt)
	ols.sendPongPktToClient(remotePeer, streamID, 0)

	done := make(chan interface{})

	go func() {
		time.Sleep(10 * time.Second)
		close(done)
	}()

loop:
	for {
		select {
		case pkt = <-recvPktCh:
			util.OverlayDebugPrintln(pkt)
			if pkt.SeqNum == 1 {
				util.OverlayDebugPrintln("OverlayServer::newHandshakeHandlingThServSide: received expected packet")
				// second packat
				break loop
			} else {
				util.OverlayDebugPrintln("OverlayServer::newHandshakeHandlingThServSide: received unexpected packet")
				// exit thread as handshake failed
				notifyErrCh <- &ClientInfo{remotePeer, streamID}
				return
			}
		case <-done:
			// exit thread without sending client info
			return
		}
	}
	ols.sendPongPktToClient(remotePeer, streamID, 1)

	finNotifyCh <- &ClientInfo{remotePeer, streamID}
}

func (ols *OverlayServer) ClientInfoNotifyPktRootHandlerTh() {
	util.OverlayDebugPrintln("OverlayServer::ClientInfoNotifyPktRootHandlerTh: start")

	// "peer name"-"stream id" -> channel to appropriate packet handling thread
	handshakePktHandleThChans := sync.Map{}
	finCh := make(chan *ClientInfo, 1)
	notifyErrCh := make(chan *ClientInfo, 1)

	for {
		util.OverlayDebugPrintln("OverlayServer::ClientInfoNotifyPktRootHandlerTh: waiting pkt")
		select {
		case pkt := <-ols.GossipMM.NotifyPktChForServerSide:
			util.OverlayDebugPrintln("OverlayServer::ClientInfoNotifyPktRootHandlerTh: received pkt:", pkt)
			// pass packet to appropriate handling thread (if not exist, spawn new handling thread)
			key := pkt.FromPeer.String() + "-" + string(pkt.StreamID)
			if ch, ok := handshakePktHandleThChans.Load(key); ok {
				ch.(chan *gossip.GossipPacket) <- pkt
			} else {
				newCh := make(chan *gossip.GossipPacket, 1)

				handshakePktHandleThChans.Store(key, newCh)
				go ols.newHandshakeHandlingThServSide(pkt.FromPeer, pkt.StreamID, newCh, finCh, notifyErrCh)
				newCh <- pkt
			}
		case finInfo := <-finCh:
			// remove channel to notify origin thread
			handshakePktHandleThChans.Delete(finInfo.RemotePeerName.String() + "-" + string(finInfo.StreamID))
			// notify new request info to Accept method
			ols.Info4OLChannelRecvCh <- finInfo
		case errRemote := <-notifyErrCh:
			// handshake failed
			handshakePktHandleThChans.Delete(errRemote.RemotePeerName.String() + "-" + string(errRemote.StreamID))
		}
	}
	util.OverlayDebugPrintln("OverlayServer::ClientInfoNotifyPktRootHandlerTh: end")
}

func (ols *OverlayServer) InitClientInfoNotifyPktRootHandlerTh() error {
	ols.Info4OLChannelRecvCh = make(chan *ClientInfo, 1)

	go ols.ClientInfoNotifyPktRootHandlerTh()

	return nil
}

func (ols *OverlayServer) Accept() (*datachannel.DataChannel, mesh.PeerName, uint16, error) {
	clientInfo := <-ols.Info4OLChannelRecvCh
	fmt.Println(clientInfo)
	olc, err := NewOverlayClient(ols.P, clientInfo.RemotePeerName, ols.GossipMM)
	if err != nil {
		panic(err)
	}

	dc, _, err2 := olc.OpenChannel(clientInfo.StreamID)
	if err2 != nil {
		panic(err2)
	}

	return dc, clientInfo.RemotePeerName, clientInfo.StreamID, nil
}

func (ols *OverlayServer) Close() error {
	// TODO: need to implement OverlayServer::Close

	return nil
}
