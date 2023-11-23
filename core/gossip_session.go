package core

import (
	"github.com/ryogrid/gossip-overlay/util"
	"math"
	"net"
	"sync"
	"time"
)

// represents a gossip session throuth remote Peer
// implementation of net.Conn
type GossipSession struct {
	LocalAddress  *PeerAddress
	RemoteAddress *PeerAddress
	SessMtx       sync.RWMutex
	// buffered data is accessed through GossipDM each time
	GossipDM    *GossipDataManager
	SessionSide OperationSideAt
}

// GossipSetton implements net.Conn
var _ net.Conn = &GossipSession{}

// Read
func (oc *GossipSession) Read(b []byte) (n int, err error) {
	util.OverlayDebugPrintln("GossipSession.Read called")
	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()

	//for oc.GossipDM.LastRecvPeer == math.MaxUint64 && oc.GossipDM.Peer.Type == Server {
	//	// no message received yet at gosship layer
	//	// so can't decide which buffer to read
	//	time.Sleep(1 * time.Millisecond)
	//}

	//util.OverlayDebugPrintln("after LastRecvPeer value check.")
	//if oc.RemoteAddress.PeerName == math.MaxUint64 {
	//	// first message at gosship layer
	//	// so set src info to oc (server side only)
	//	util.OverlayDebugPrintln("Set Remote PeerName.")
	//	oc.RemoteAddress.PeerName = oc.GossipDM.LastRecvPeer
	//}

	var buf []byte
	if oc.SessionSide == ServerSide {
		buf = oc.GossipDM.Read(math.MaxUint64, ServerSide)
	} else if oc.SessionSide == ClientSide {
		buf = oc.GossipDM.Read(oc.RemoteAddress.PeerName, ClientSide)
	} else {
		panic("invalid SessionSide")
	}

	//if oc.RemoteAddress.PeerName == math.MaxUint64 {
	//	// Assosiation passed this GossipSession has not created Stream yet (server side)
	//	buf = oc.GossipDM.Read(math.MaxUint64)
	//} else {
	//	buf = oc.GossipDM.Read(oc.RemoteAddress.PeerName)
	//}

	//ret := make([]byte, len(buf))
	//copy(ret, buf)
	//b = ret
	copy(b, buf)

	return len(buf), nil

	//i, rAddr, err := oc.pConn.ReadFrom(p)
	//if err != nil {
	//	return 0, err
	//}
	//oc.SessMtx.Lock()
	//oc.rAddr = rAddr
	//oc.SessMtx.Unlock()
	//
	//return i, err
}

// Write writes len(p) bytes from p to the DTLS connection
func (oc *GossipSession) Write(b []byte) (n int, err error) {
	util.OverlayDebugPrintln("GossipSession.Write called", b)

	////return oc.pConn.WriteTo(p, oc.RemoteAddr())
	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()
	//oc.GossipDM.Write(oc.RemoteAddress.PeerName, b)
	//return len(b), nil

	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()

	//oc.GossipDM.SendToRemote(b)
	oc.GossipDM.SendToRemote(oc.RemoteAddress.PeerName, b)

	return len(b), nil
}

// Close closes the conn and releases any Read calls
func (oc *GossipSession) Close() error {
	//return oc.pConn.WhenClose()
	oc.GossipDM.WhenClose(oc.RemoteAddress.PeerName)
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
	util.OverlayDebugPrintln("GossipSession.RemoteAddr called", oc.RemoteAddress.PeerName)
	if oc.RemoteAddress != nil {
		return oc.RemoteAddress
	}
	return nil
	//oc.SessMtx.RLock()
	//defer oc.SessMtx.RUnlock()
	//return oc.RemoteAddress
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
