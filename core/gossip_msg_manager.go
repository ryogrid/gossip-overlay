package core

import (
	"fmt"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"sync"
	"time"
)

type GossipMessageManager struct {
	LocalAddress *PeerAddress
	GossipDM     *GossipDataManager
	//// "<peer name>-<stream id" => channel to appropriate packet handling thread
	PktHandlers    map[string]chan *GossipPacket
	PktHandlersMtx sync.Mutex
	Actions        chan<- func()
	Quit           chan struct{}
	// first notify packet from client is sent to this channel
	NotifyPktChForServerSide chan *GossipPacket
}

func NewGossipMessageManager(localAddress *PeerAddress, gossipDM *GossipDataManager) *GossipMessageManager {
	actions := make(chan func())
	ret := &GossipMessageManager{
		LocalAddress:             localAddress,
		GossipDM:                 gossipDM,
		PktHandlers:              make(map[string]chan *GossipPacket),
		PktHandlersMtx:           sync.Mutex{},
		Actions:                  actions,
		Quit:                     make(chan struct{}),
		NotifyPktChForServerSide: nil,
	}

	go ret.loop(actions)

	return ret
}

func (gmm *GossipMessageManager) loop(actions <-chan func()) {
	for {
		select {
		case f := <-actions:
			f()
		case <-gmm.Quit:
			return
		}
	}
}

func (gmm *GossipMessageManager) Stop() {
	close(gmm.Quit)
}

func (gmm *GossipMessageManager) RegisterChToHandlerTh(dest mesh.PeerName, streamID uint16, recvPktCh chan *GossipPacket) {
	// TODO: need to implement (GossipMessageManager::RegisterChToHandlerTh)
	panic("not implemented")
}

func (gmm *GossipMessageManager) UnregisterChToHandlerTh(dest mesh.PeerName, streamID uint16, recvPktCh chan *GossipPacket) {
	// TODO: need to implement (GossipMessageManager::UnregisterChToHandlerTh)
	panic("not implemented")
}

func (gmm *GossipMessageManager) SendToRemote(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)
	c := make(chan struct{})
	gmm.Actions <- func() {
		defer close(c)
		if gmm.GossipDM.Peer.Send != nil {
			sendObj := GossipPacket{
				FromPeer:     gmm.LocalAddress.PeerName,
				Buf:          data,
				ReceiverSide: recvOpSide,
				StreamID:     streamID,
				// TODO: need to set proper value of GossipPacket(GossipMessageManager::SendToRemote)
			}
			encodedData := sendObj.Encode()[0]
			for {
				err := gmm.GossipDM.Peer.Send.GossipUnicast(dest, encodedData)
				if err == nil {
					break
				} else {
					// TODO: need to implement timeout
					util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: err:", err)
					util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: 1sec wait and do retry")
					time.Sleep(1 * time.Second)
				}
			}
		} else {
			gmm.GossipDM.Peer.Logger.Printf("no sender configured; not broadcasting update right now")
		}
	}
	<-c

	return nil
}

// use at notification of information for CtoC stream establishment (4way handshake)
// and at doing heartbeat (maybe)
func (gmm *GossipMessageManager) SendPingAndWaitPong(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)
	// TODO: need to implement timeout (GossipMessageManager::SendPingAndWaitPong)

	c := make(chan struct{})
	go func() {
		defer close(c)
		var recvPktCh chan *GossipPacket
		if gmm.GossipDM.Peer.Send != nil {
			recvPktCh = make(chan *GossipPacket)
			gmm.RegisterChToHandlerTh(dest, streamID, recvPktCh)
			gmm.SendToRemote(dest, streamID, recvOpSide, data)
		} else {
			panic("no sender configured; not broadcasting update right now")
		}

		pkt := <-recvPktCh
		// TODO: need to check received pkt content is appropriate (GossipMessageManager::SendPingAndWaitPong)
		fmt.Println(pkt)
		panic("not implemented")
	}()
	<-c

	panic("not implemented")
}

// called when any packet received (even if packat is of SCTP CtoC stream)
func (gmm *GossipMessageManager) OnPacketReceived(src mesh.PeerName, buf []byte) error {
	gp, err := DecodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}

	if gp.PktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.ReceiverSide == ServerSide {
		gmm.NotifyPktChForServerSide <- gp
		return nil
	} else if gp.PktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.ReceiverSide == ClientSide {
		gmm.PktHandlersMtx.Lock()
		destCh := gmm.PktHandlers[gp.FromPeer.String()+"-"+string(gp.StreamID)]
		gmm.PktHandlersMtx.Unlock()
		if destCh != nil {
			destCh <- gp
		} else {
			panic("illigal internal state!")
		}
	}

	// when packat is of CtoC stream
	err2 := gmm.GossipDM.Peer.GossipDataMan.WriteToLocalBuffer(gmm.GossipDM.Peer, src, gp.StreamID, gp.ReceiverSide, gp.Buf)
	if err2 != nil {
		panic(err2)
	}
	util.OverlayDebugPrintln(fmt.Sprintf("OnPacketReceived %s %v", src, buf))
	return nil
}

func (gmm *GossipMessageManager) WhenClose(remotePeer mesh.PeerName, streamID uint16) error {
	util.OverlayDebugPrintln("GossipDMessageManager.WhenClose called. remotePeer:", remotePeer, " streamID:", streamID)
	gmm.GossipDM.RemoveBuffer(remotePeer, streamID)

	return nil
}
