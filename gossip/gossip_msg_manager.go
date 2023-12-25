package gossip

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
		NotifyPktChForServerSide: make(chan *GossipPacket),
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
		if gmm.GossipDM.peer.send != nil {
			pktKind := PACKET_KIND_NOTIFY_PEER_INFO
			if seqNum == math.MaxUint64 {
				pktKind = PACKET_KIND_CTC_DATA
			}
			sendObj := GossipPacket{
				FromPeer:     gmm.LocalAddress.PeerName,
				buf:          data,
				receiverSide: recvOpSide,
				StreamID:     streamID,
				SeqNum:       seqNum,
				pktKind:      pktKind,
			}
			encodedData := sendObj.Encode()[0]
			util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: encodedData:", encodedData)
			for {
				err := gmm.GossipDM.peer.send.GossipUnicast(dest, encodedData)
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
			gmm.GossipDM.peer.logger.Printf("no sender configured; not broadcasting update right now")
		}
	}
	<-c

	return nil
}

// use at notification of information for CtoC stream establishment (4way handshake)
// and at doing heartbeat (maybe)
func (gmm *GossipMessageManager) SendPingAndWaitPong(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, timeout time.Duration, seqNum uint64, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendPingAndPong called. dest:", dest, "streamID:", streamID, " data:", data)

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
		if gmm.GossipDM.peer.send != nil {
			recvPktCh = make(chan *GossipPacket)
			gmm.RegisterChToHandlerTh(dest, streamID, recvPktCh)
			gmm.GossipDM.bufs.Store(dest.String()+"-"+string(streamID), make([]byte, 0))
			gmm.SendToRemote(dest, streamID, recvOpSide, seqNum, data)
		} else {
			panic("no sender configured; not broadcasting update right now")
		}

	loop:
		for {
			select {
			case <-recvPktCh:
				util.OverlayDebugPrintln("GossipMessageManager.SendPingAndPong: received pong packet")
				gmm.UnregisterChToHandlerTh(dest, streamID, recvPktCh)
				ret = nil
				break loop
			case <-done:
				gmm.UnregisterChToHandlerTh(dest, streamID, recvPktCh)
				ret = errors.New("timeout reached")
				break loop
			}
		}
	}()
	gmm.GossipDM.bufs.Delete(dest.String() + "-" + string(streamID))
	<-c
	return ret
}

// called when any packet received (even if packat is of SCTP CtoC stream)
func (gmm *GossipMessageManager) OnPacketReceived(src mesh.PeerName, buf []byte) error {
	gp, err := decodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}
	util.OverlayDebugPrintln("GossipMessageManager.OnPacketReceived called. src:", src, " streamId:", gp.StreamID, " buf:", buf)

	if gp.pktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.receiverSide == ServerSide {
		util.OverlayDebugPrintln("GossipMessageManager.OnPacketReceived: notify packet received and passed to root handler (ServerSide)")
		gmm.NotifyPktChForServerSide <- gp
		util.OverlayDebugPrintln("GossipMessageManager.OnPacketReceived: send to root handler finished (ServerSide)")
		return nil
	} else if gp.pktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.receiverSide == ClientSide {
		util.OverlayDebugPrintln("GossipMessageManager.OnPacketReceived: notify packet received and passed to each handler (ClientSide)")
		destCh, ok := gmm.PktHandlers.Load(gp.FromPeer.String() + "-" + string(gp.StreamID))
		if ok {
			destCh.(chan *GossipPacket) <- gp
			return nil
		} else {
			panic("illigal internal state!")
		}
	}

	// when packat is of CtoC stream
	err2 := gmm.GossipDM.write(gp.FromPeer, gp.StreamID, gp.buf)
	if err2 != nil {
		panic(err2)
	}
	util.OverlayDebugPrintln(fmt.Sprintf("OnPacketReceived %s %v", src, buf))
	return nil
}

func (gmm *GossipMessageManager) WhenClose(remotePeer mesh.PeerName, streamID uint16) error {
	util.OverlayDebugPrintln("GossipDMessageManager.WhenClose called. remotePeer:", remotePeer, " streamID:", streamID)
	gmm.GossipDM.removeBuffer(remotePeer, streamID)

	return nil
}
