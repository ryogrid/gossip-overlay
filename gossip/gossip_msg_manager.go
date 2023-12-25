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
	localAddress *PeerAddress
	gossipDM     *GossipDataManager
	//// "<peer name>-<stream id" => channel to appropriate packet handling thread
	pktHandlers sync.Map
	actions     chan<- func()
	quit        chan struct{}
	// notification packet from client is sent to this channel
	NotifyPktChForServerSide chan *GossipPacket
}

func NewGossipMessageManager(localAddress *PeerAddress, gossipDM *GossipDataManager) *GossipMessageManager {
	actions := make(chan func())
	ret := &GossipMessageManager{
		localAddress:             localAddress,
		gossipDM:                 gossipDM,
		pktHandlers:              sync.Map{},
		actions:                  actions,
		quit:                     make(chan struct{}),
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
		case <-gmm.quit:
			return
		}
	}
}

func (gmm *GossipMessageManager) stop() {
	close(gmm.quit)
}

func (gmm *GossipMessageManager) registerChToHandlerTh(dest mesh.PeerName, streamID uint16, recvPktCh chan *GossipPacket) {
	gmm.pktHandlers.Store(dest.String()+"-"+string(streamID), recvPktCh)
}

func (gmm *GossipMessageManager) unregisterChToHandlerTh(dest mesh.PeerName, streamID uint16, recvPktCh chan *GossipPacket) {
	gmm.pktHandlers.Delete(dest.String() + "-" + string(streamID))
}

func (gmm *GossipMessageManager) SendToRemote(dest mesh.PeerName, streamID uint16, recvOpSide OperationSideAt, seqNum uint64, data []byte) error {
	util.OverlayDebugPrintln("GossipMessageManager.SendToRemote called. dest:", dest, "streamID:", streamID, " data:", data)
	c := make(chan struct{})
	gmm.actions <- func() {
		defer close(c)
		if gmm.gossipDM.peer.send != nil {
			pktKind := PACKET_KIND_NOTIFY_PEER_INFO
			if seqNum == math.MaxUint64 {
				pktKind = PACKET_KIND_CTC_DATA
			}
			sendObj := GossipPacket{
				FromPeer:     gmm.localAddress.PeerName,
				buf:          data,
				receiverSide: recvOpSide,
				StreamID:     streamID,
				SeqNum:       seqNum,
				pktKind:      pktKind,
			}
			encodedData := sendObj.Encode()[0]
			util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: encodedData:", encodedData)
			for {
				err := gmm.gossipDM.peer.send.GossipUnicast(dest, encodedData)
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
			gmm.gossipDM.peer.logger.Printf("no sender configured; not broadcasting update right now")
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
		if gmm.gossipDM.peer.send != nil {
			recvPktCh = make(chan *GossipPacket)
			gmm.registerChToHandlerTh(dest, streamID, recvPktCh)
			gmm.gossipDM.bufs.Store(dest.String()+"-"+string(streamID), make([]byte, 0))
			gmm.SendToRemote(dest, streamID, recvOpSide, seqNum, data)
		} else {
			panic("no sender configured; not broadcasting update right now")
		}

	loop:
		for {
			select {
			case <-recvPktCh:
				util.OverlayDebugPrintln("GossipMessageManager.SendPingAndPong: received pong packet")
				gmm.unregisterChToHandlerTh(dest, streamID, recvPktCh)
				ret = nil
				break loop
			case <-done:
				gmm.unregisterChToHandlerTh(dest, streamID, recvPktCh)
				ret = errors.New("timeout reached")
				break loop
			}
		}
	}()
	gmm.gossipDM.bufs.Delete(dest.String() + "-" + string(streamID))
	<-c
	return ret
}

// called when any packet received (even if packat is of SCTP CtoC stream)
func (gmm *GossipMessageManager) onPacketReceived(src mesh.PeerName, buf []byte) error {
	gp, err := decodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}
	util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived called. src:", src, " streamId:", gp.StreamID, " buf:", buf)

	if gp.pktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.receiverSide == ServerSide {
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: notify packet received and passed to root handler (ServerSide)")
		gmm.NotifyPktChForServerSide <- gp
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: send to root handler finished (ServerSide)")
		return nil
	} else if gp.pktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.receiverSide == ClientSide {
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: notify packet received and passed to each handler (ClientSide)")
		destCh, ok := gmm.pktHandlers.Load(gp.FromPeer.String() + "-" + string(gp.StreamID))
		if ok {
			destCh.(chan *GossipPacket) <- gp
			return nil
		} else {
			panic("illigal internal state!")
		}
	}

	// when packat is of CtoC stream
	err2 := gmm.gossipDM.write(gp.FromPeer, gp.StreamID, gp.buf)
	if err2 != nil {
		panic(err2)
	}
	util.OverlayDebugPrintln(fmt.Sprintf("onPacketReceived %s %v", src, buf))
	return nil
}

func (gmm *GossipMessageManager) whenClose(remotePeer mesh.PeerName, streamID uint16) error {
	util.OverlayDebugPrintln("GossipDMessageManager.whenClose called. remotePeer:", remotePeer, " streamID:", streamID)
	gmm.gossipDM.removeBuffer(remotePeer, streamID)

	return nil
}