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
	var ret error = nil
	c := make(chan struct{})
	gmm.actions <- func() {
		defer close(c)
		if gmm.gossipDM.peer.send != nil {
			pktKind := PACKET_KIND_NOTIFY_PEER_INFO
			if seqNum == math.MaxUint64 {
				pktKind = PACKET_KIND_CTC_DATA
			} else if seqNum == math.MaxUint32 {
				pktKind = PACKET_KIND_CTC_HEARTBEAT
			}

			sendObj := GossipPacket{
				FromPeer:     gmm.localAddress.PeerName,
				Buf:          data,
				ReceiverSide: recvOpSide,
				StreamID:     streamID,
				SeqNum:       seqNum,
				PktKind:      pktKind,
			}
			encodedData := sendObj.Encode()[0]
			util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: encodedData:", encodedData)
			for {
				err := gmm.gossipDM.peer.send.GossipUnicast(dest, encodedData)
				if err == nil {
					break
				} else {
					if pktKind == PACKET_KIND_CTC_HEARTBEAT {
						ret = err
						break
					} else {
						// TODO: need to implement timeout
						util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: err:", err)
						util.OverlayDebugPrintln("GossipMessageManager.SendToRemote: 1sec wait and do retry")
						time.Sleep(1 * time.Second)
					}
				}
			}
		} else {
			gmm.gossipDM.peer.logger.Printf("no sender configured; not broadcasting update right now")
		}
	}
	<-c

	return ret
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
			//gmm.gossipDM.bufs.Store(dest.String()+"-"+string(streamID), make([]byte, 0))
			err := gmm.SendToRemote(dest, streamID, recvOpSide, seqNum, data)
			if err != nil {
				ret = errors.New("remote peer becomes not available")
				return
			}
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
	//gmm.gossipDM.bufs.Delete(dest.String() + "-" + string(streamID))
	<-c
	return ret
}

// if seqNum is math.MaxUint32, it means that this packet is for heartbeat
// if seqNum is other, it means that this packet is for notification of information for CtoC stream establishment (4way handshake)
func (gmm *GossipMessageManager) SendPongPktToClient(remotePeer mesh.PeerName, streamID uint16, seqNum uint64) error {
	err := gmm.SendToRemote(remotePeer, streamID, ClientSide, seqNum, []byte{})
	if err != nil {
		//fmt.Println(err)
		//return err
		panic(err)
	}
	return nil
}

func (gmm *GossipMessageManager) handleKeepAlivePkt(remotePeer mesh.PeerName, streamID uint16, pkt *GossipPacket) {
	util.OverlayDebugPrintln("GossipMessageManager.handleKeepAlivePkt called. remotePeer:", remotePeer, " streamID:", streamID, " pkt:", pkt)
	util.OverlayDebugPrintln(pkt)

	gmm.SendPongPktToClient(remotePeer, streamID, 0)
}

// called when any packet received (even if packat is of SCTP CtoC stream)
func (gmm *GossipMessageManager) onPacketReceived(src mesh.PeerName, buf []byte) error {
	gp, err := decodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}
	util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived called. src:", src, " streamId:", gp.StreamID, " Buf:", buf)
	util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived called. gp:", *gp)

	if gp.PktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.ReceiverSide == ServerSide {
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: notify packet received and passed to root handler (ServerSide)")
		gmm.NotifyPktChForServerSide <- gp
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: send to root handler finished (ServerSide)")
		return nil
	} else if gp.PktKind == PACKET_KIND_NOTIFY_PEER_INFO && gp.ReceiverSide == ClientSide {
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: notify packet received and passed to each handler (ClientSide)")
		destCh, ok := gmm.pktHandlers.Load(gp.FromPeer.String() + "-" + string(gp.StreamID))
		if ok {
			destCh.(chan *GossipPacket) <- gp
			return nil
		} else {
			panic("illigal internal state!")
		}
	} else if gp.PktKind == PACKET_KIND_CTC_HEARTBEAT && gp.ReceiverSide == ClientSide { // heartbeat
		util.OverlayDebugPrintln("GossipMessageManager.onPacketReceived: heartbeat packet received")
		go gmm.handleKeepAlivePkt(gp.FromPeer, gp.StreamID, gp)
	}

	// when packat is of CtoC stream
	err2 := gmm.gossipDM.write(gp.FromPeer, gp.StreamID, gp.Buf)
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

func (gmm *GossipMessageManager) NewGossipSessionForClientToClient(remotePeer mesh.PeerName, streamID uint16) (*GossipSession, error) {
	ret := &GossipSession{
		localAddress: &PeerAddress{gmm.gossipDM.Self},
		//remoteAddress:      []*PeerAddress{&PeerAddress{remotePeer}},
		remoteAddress: &PeerAddress{remotePeer},
		//RemoteAddressesMtx: &sync.Mutex{},
		//SessMtx:            sync.RWMutex{},
		gossipDM:          gmm.gossipDM,
		remoteSessionSide: ClientSide,
		StreamID:          streamID,
		IsActive:          true,
	}
	if _, ok := gmm.gossipDM.loadBuffer(remotePeer, streamID); !ok {
		gmm.gossipDM.storeBuffer(remotePeer, streamID, NewBufferWithMutex(make([]byte, 0)))
	}

	return ret, nil
}
