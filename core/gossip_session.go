package core

import (
	"errors"
	"github.com/ryogrid/gossip-overlay/util"
	"math"
	"net"
	"time"
)

// represents a gossip session throuth remote Peer
// implementation of net.Conn
type GossipSession struct {
	LocalAddress  *PeerAddress
	RemoteAddress *PeerAddress
	//RemoteAddressesMtx *sync.Mutex
	//SessMtx sync.RWMutex
	// buffered data is accessed through GossipDM each time
	GossipDM          *GossipDataManager
	LocalSessionSide  OperationSideAt
	RemoteSessionSide OperationSideAt
	StreamID          uint16
}

// GossipSetton implements net.Conn
var _ net.Conn = &GossipSession{}

// Read
func (oc *GossipSession) Read(b []byte) (n int, err error) {
	util.OverlayDebugPrintln("GossipSession.Read called")

	var buf []byte
	if oc.LocalSessionSide == ServerSide {
		buf = oc.GossipDM.Read(math.MaxUint64, 0, ServerSide)
	} else if oc.LocalSessionSide == ClientSide {
		//oc.RemoteAddressesMtx.Lock()
		//peerName := oc.RemoteAddress[0].PeerName
		peerName := oc.RemoteAddress.PeerName
		//oc.RemoteAddressesMtx.Unlock()
		buf = oc.GossipDM.Read(peerName, oc.StreamID, ClientSide)
		if buf == nil {
			return 0, errors.New("session closed")
		}
	} else {
		panic("invalid LocalSessionSide")
	}
	copy(b, buf)

	return len(buf), nil
}

// Write writes len(p) bytes from p to the DTLS connection
func (oc *GossipSession) Write(b []byte) (n int, err error) {
	util.OverlayDebugPrintln("GossipSession.Write called", b)

	//if len(oc.RemoteAddress) > 0 {
	if oc.LocalSessionSide == ClientSide {
		//peerNames := make([]mesh.PeerName, 0)
		//oc.RemoteAddressesMtx.Lock()
		//for _, remoteAddr := range oc.RemoteAddress {
		//	peerNames = append(peerNames, remoteAddr.PeerName)
		//}
		//oc.RemoteAddressesMtx.Unlock()
		//
		//// when client side, length of peerNames is 1, so send data to only one stream
		//// but when server side, length of peerNames is same as number of established streams
		//// server side sends same data to all streams (it is ridiculous...)
		//// because collect destination decidable design can't be implemented with mesh lib
		//// stream collectly must work even if send data to not collect destination
		//// by control of SCTP protocol layer...
		//for _, peerName := range peerNames {
		//	oc.GossipDM.SendToRemote(peerName, oc.LocalSessionSide, b)
		//}
		if oc.GossipDM.IsInjectPacketLoss {
			// drop packet
			oc.GossipDM.IsInjectPacketLoss = false
			return len(b), nil
		}
		oc.GossipDM.SendToRemote(oc.RemoteAddress.PeerName, oc.StreamID, oc.RemoteSessionSide, b)
	} else if oc.LocalSessionSide == ServerSide {
		// server side uses LastRecvPeer until StreamToNotifySelfInfo is established
		// because remote peer name can't be known until then
		for oc.GossipDM.LastRecvPeer == math.MaxUint64 {
			time.Sleep(100 * time.Millisecond)
		}
		//oc.GossipDM.SendToRemote(oc.GossipDM.LastRecvPeer, oc.LocalSessionSide, b)
		oc.GossipDM.SendToRemote(oc.GossipDM.LastRecvPeer, oc.StreamID, oc.RemoteSessionSide, b)
	} else {
		panic("invalid LocalSessionSide")
	}

	return len(b), nil
}

// Close closes the conn and releases any Read calls
func (oc *GossipSession) Close() error {
	oc.GossipDM.WhenClose(oc.RemoteAddress.PeerName, oc.StreamID)

	return nil
}

func (oc *GossipSession) LocalAddr() net.Addr {
	util.OverlayDebugPrintln("GossipSession.LocalAddr called", oc.LocalAddress.PeerName)
	if oc.LocalAddress != nil {
		return oc.LocalAddress
	}
	return nil
}

func (oc *GossipSession) RemoteAddr() net.Addr {
	util.OverlayDebugPrintln("GossipSession.RemoteAddr called")
	//oc.RemoteAddressesMtx.Lock()
	//defer oc.RemoteAddressesMtx.Unlock()
	//if len(oc.RemoteAddress) > 0 {
	//	return oc.RemoteAddress[0]
	//}
	if oc.RemoteAddress.PeerName != math.MaxUint64 {
		return oc.RemoteAddress
	}
	return nil
}

// SetDeadline is a stub
func (oc *GossipSession) SetDeadline(t time.Time) error {
	util.OverlayDebugPrintln("GossipSession.SetDeadline called", t)
	return nil
}

// SetReadDeadline is a stub
func (oc *GossipSession) SetReadDeadline(t time.Time) error {
	util.OverlayDebugPrintln("GossipSession.SetReadDeadline called", t)
	return nil
}

// SetWriteDeadline is a stub
func (oc *GossipSession) SetWriteDeadline(t time.Time) error {
	util.OverlayDebugPrintln("GossipSession.SetWriteDeadline called", t)
	return nil
}
