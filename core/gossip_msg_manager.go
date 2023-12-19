package core

import (
	"errors"
	"fmt"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math"
	"sync"
	"time"
)

type GossipMessageManager struct {
	LocalAddress *PeerAddress
	GossipDM     *GossipDataManager
	//// "<peer name>-<stream id" => channel to appropriate packet handling thread
	PktHandlers sync.Map
	Actions     chan<- func()
	Quit        chan struct{}
	// notification packet from client is sent to this channel
	NotifyPktChForServerSide chan *GossipPacket
}

func NewGossipMessageManager(localAddress *PeerAddress, gossipDM *GossipDataManager) *GossipMessageManager {
	actions := make(chan func())
	ret := &GossipMessageManager{
		LocalAddress:             localAddress,
		GossipDM:                 gossipDM,
		PktHandlers:              sync.Map{},
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
	gmm.PktHandlers.Store(dest.String()+"-"+string(streamID), recvPktCh)
}

func (gmm *GossipMessageManager) UnregisterChToHandlerTh(dest mesh.PeerName, streamID uint16, recvPktCh chan *GossipPacket) {
	gmm.PktHandlers.Delete(dest.String() + "-" + string(streamID))
}

func (gmm *GossipMessageManager) SendToRemote(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, seqNum uint64, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)
	c := make(chan struct{})
	gmm.Actions <- func() {
		defer close(c)
		if gmm.GossipDM.Peer.Send != nil {
			pktKind := PACKET_KIND_NOTIFY_PEER_INFO
			if seqNum == math.MaxUint64 {
				pktKind = PACKET_KIND_CTC_DATA
			}
			sendObj := GossipPacket{
				FromPeer:     gmm.LocalAddress.PeerName,
				Buf:          data,
				ReceiverSide: recvOpSide,
				StreamID:     streamID,
				SeqNum:       seqNum,
				PktKind:      pktKind,
			}
			encodedData := sendObj.Encode()[0]
			util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: encodedData:", encodedData)
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
func (gmm *GossipMessageManager) SendPingAndWaitPong(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, timeout time.Duration, seqNum uint64, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)

	var ret error
	c := make(chan struct{})
	done := make(chan interface{})

	// timeout
	go func() {
		time.Sleep(timeout)
		close(done)
	}()

	go func() {
		defer close(c)
		var recvPktCh chan *GossipPacket
		if gmm.GossipDM.Peer.Send != nil {
			recvPktCh = make(chan *GossipPacket)
			gmm.RegisterChToHandlerTh(dest, streamID, recvPktCh)
			gmm.SendToRemote(dest, streamID, recvOpSide, seqNum, data)
		} else {
			panic("no sender configured; not broadcasting update right now")
		}

	loop:
		for {
			select {
			case <-recvPktCh:
				gmm.UnregisterChToHandlerTh(dest, streamID, recvPktCh)
				ret = nil
				break loop
			case <-done:
				ret = errors.New("timeout reached")
				break loop
			}
		}
	}()
	<-c
	return ret
}

// called when any packet received (even if packat is of SCTP CtoC stream)
func (gmm *GossipMessageManager) OnPacketReceived(src mesh.PeerName, buf []byte) error {
	gp, err := DecodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}
	util.OverlayDebugPrintln("GossipMessageManager.OnPacketReceived called. src:", src, " streamId:", gp.StreamID, " buf:", buf)

	if gp.PktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.ReceiverSide == ServerSide {
		gmm.NotifyPktChForServerSide <- gp
		return nil
	} else if gp.PktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.ReceiverSide == ClientSide {

		destCh, ok := gmm.PktHandlers.Load(gp.FromPeer.String() + "-" + string(gp.StreamID))
		if ok {
			destCh.(chan *GossipPacket) <- gp
			return nil
		} else {
			panic("illigal internal state!")
		}
	}

	// when packat is of CtoC stream
	err2 := gmm.GossipDM.Write(gp.FromPeer, gp.StreamID, gp.Buf)
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
