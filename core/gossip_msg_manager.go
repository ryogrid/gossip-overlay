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
	// "<peer name>-<stream id" => channel to appropriate packet handling thread
	PktHandlers    map[string]chan *GossipPacket
	PktHandlersMtx sync.Mutex
}

func NewGossipMessageManager(localAddress *PeerAddress, gossipDM *GossipDataManager) *GossipMessageManager {
	return &GossipMessageManager{
		LocalAddress:   localAddress,
		GossipDM:       gossipDM,
		PktHandlers:    make(map[string]chan *GossipPacket),
		PktHandlersMtx: sync.Mutex{},
	}
}

func (gmm *GossipMessageManager) RegisterChToHandlerTh(chan *GossipPacket) {
	// TODO: need to implement (GossipMessageManager::RegisterChToHandlerTh)
	panic("not implemented")
}

func (gmm *GossipMessageManager) UnregisterChToHandlerTh(chan *GossipPacket) {
	// TODO: need to implement (GossipMessageManager::RegisterChToHandlerTh)
	panic("not implemented")
}

func (gmm *GossipMessageManager) SendToRemote(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)
	c := make(chan struct{})
	gmm.GossipDM.Peer.Actions <- func() {
		defer close(c)
		if gmm.GossipDM.Peer.Send != nil {
			sendObj := GossipPacket{
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

func (gmm *GossipMessageManager) OnPacketReceived(src mesh.PeerName, buf []byte) error {
	gp, err := DecodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}

	// TODO: need to handle packets according to its kind (Peer::OnPacketReceived)

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
