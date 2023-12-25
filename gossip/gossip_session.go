package gossip

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
	localAddress  *PeerAddress
	remoteAddress *PeerAddress
	// buffered data is accessed through gossipDM each time
	gossipDM          *GossipDataManager
	remoteSessionSide OperationSideAt
	StreamID          uint16
}

// GossipSetton implements net.Conn
var _ net.Conn = &GossipSession{}

// Read
func (oc *GossipSession) Read(b []byte) (n int, err error) {
	util.OverlayDebugPrintln("GossipSession.read called")

	var buf []byte
	peerName := oc.remoteAddress.PeerName
	buf = oc.gossipDM.read(peerName, oc.StreamID)
	if buf == nil {
		return 0, errors.New("session closed")
	}
	copy(b, buf)

	return len(buf), nil
}

// Write writes len(p) bytes from p to the DTLS connection
func (oc *GossipSession) Write(b []byte) (n int, err error) {
	util.OverlayDebugPrintln("GossipSession.write called", b)

	oc.gossipDM.peer.GossipMM.SendToRemote(oc.remoteAddress.PeerName, oc.StreamID, oc.remoteSessionSide, math.MaxUint64, b)

	return len(b), nil
}

// Close closes the conn and releases any Read calls
func (oc *GossipSession) Close() error {
	oc.gossipDM.peer.GossipMM.whenClose(oc.remoteAddress.PeerName, oc.StreamID)

	return nil
}

func (oc *GossipSession) LocalAddr() net.Addr {
	util.OverlayDebugPrintln("GossipSession.LocalAddr called", oc.localAddress.PeerName)
	if oc.localAddress != nil {
		return oc.localAddress
	}
	return nil
}

func (oc *GossipSession) RemoteAddr() net.Addr {
	util.OverlayDebugPrintln("GossipSession.RemoteAddr called")
	if oc.remoteAddress.PeerName != math.MaxUint64 {
		return oc.remoteAddress
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
