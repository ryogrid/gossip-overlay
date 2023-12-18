package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pion/datachannel"
	"github.com/weaveworks/mesh"
	"sync"
)

type ClientInfo struct {
	RemotePeerName mesh.PeerName
	StreamID       uint16
}

// wrapper of sctp.Server
type OverlayServer struct {
	P                    *Peer
	Info4OLChannelRecvCh chan *ClientInfo
	GossipMM             *GossipMessageManager
}

func NewOverlayServer(p *Peer, gossipMM *GossipMessageManager) (*OverlayServer, error) {
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

func (ols *OverlayServer) sendPongPktToClient(remotePeer mesh.PeerName, streamID uint16) error {
	err := ols.GossipMM.SendToRemote(remotePeer, streamID, ClientSide, []byte{})
	if err != nil {
		fmt.Println(err)
		return err
	}
}

// thread for handling client info notify packet of each client
func (ols *OverlayServer) newHandshakeHandlingThServSide(remotePeer mesh.PeerName, streamID uint16,
	recvPktCh chan *GossipPacket, finNotifyCh chan *ClientInfo) {
	// TODO: need to initialization

	// TODO: need to implement (OverlayServer::newHandshakeHandlingTh)
	//       - recv ping and send pong packet twice
	//       - when ping-pong x2 is finished, send client info to ols.Info4OLChannelRecvCh channel
	pkt := <-recvPktCh
	// checking of first packet is not needed
	fmt.Println(pkt)

	finNotifyCh <- &ClientInfo{remotePeer, streamID}
}

func (ols *OverlayServer) ClientInfoNotifyPktRootHandlerTh() {
	// "peer name"-"stream id" -> channel to appropriate packet handling thread
	handshakePktHandleThChans := sync.Map{}
	finCh := make(chan *ClientInfo, 1)

	for {
		select {
		case pkt := <-ols.GossipMM.NotifyPktChForServerSide:
			// pass packet to appropriate handling thread (if not exist, spawn new handling thread)
			key := pkt.FromPeer.String() + "-" + string(pkt.StreamID)
			if ch, ok := handshakePktHandleThChans.Load(key); ok {
				ch.(chan *GossipPacket) <- pkt
			} else {
				newCh := make(chan *GossipPacket, 1)

				handshakePktHandleThChans.Store(key, newCh)
				go ols.newHandshakeHandlingThServSide(pkt.FromPeer, pkt.StreamID, newCh, finCh)
				newCh <- pkt
			}
		case finInfo := <-finCh:
			// remove channel to notify origin thread
			handshakePktHandleThChans.Delete(finInfo.RemotePeerName.String() + "-" + string(finInfo.StreamID))
			// notify new request info to Accept method
			ols.Info4OLChannelRecvCh <- finInfo
		}
	}
}

func (ols *OverlayServer) InitClientInfoNotifyPktRootHandlerTh() error {
	ols.Info4OLChannelRecvCh = make(chan *ClientInfo, 1)

	go ols.ClientInfoNotifyPktRootHandlerTh()

	return nil
}

func (ols *OverlayServer) Accept() (*datachannel.DataChannel, mesh.PeerName, uint16, error) {
	clientInfo := <-ols.Info4OLChannelRecvCh
	fmt.Println(clientInfo)
	// TODO: need to implement (OverlayServer::Accept)
	//       - establish CtoC data channel
	panic("not implemented")
}

func (ols *OverlayServer) Close() error {
	// TODO: need to implement OverlayServer::Close

	return nil
}
