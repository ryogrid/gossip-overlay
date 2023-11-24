package core

import (
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"math"
	"net"
	"sync"
	"time"
)

// represents a gossip session throuth remote Peer
// implementation of net.Conn
type GossipSession struct {
	LocalAddress       *PeerAddress
	RemoteAddresses    []*PeerAddress
	RemoteAddressesMtx *sync.Mutex
	SessMtx            sync.RWMutex
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
	//if oc.RemoteAddresses.PeerName == math.MaxUint64 {
	//	// first message at gosship layer
	//	// so set src info to oc (server side only)
	//	util.OverlayDebugPrintln("Set Remote PeerName.")
	//	oc.RemoteAddresses.PeerName = oc.GossipDM.LastRecvPeer
	//}

	var buf []byte
	if oc.SessionSide == ServerSide {
		buf = oc.GossipDM.Read(math.MaxUint64, ServerSide)
	} else if oc.SessionSide == ClientSide {
		oc.RemoteAddressesMtx.Lock()
		peerName := oc.RemoteAddresses[0].PeerName
		oc.RemoteAddressesMtx.Unlock()
		buf = oc.GossipDM.Read(peerName, ClientSide)
	} else {
		panic("invalid SessionSide")
	}

	//if oc.RemoteAddresses.PeerName == math.MaxUint64 {
	//	// Assosiation passed this GossipSession has not created Stream yet (server side)
	//	buf = oc.GossipDM.Read(math.MaxUint64)
	//} else {
	//	buf = oc.GossipDM.Read(oc.RemoteAddresses.PeerName)
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
	//oc.GossipDM.Write(oc.RemoteAddresses.PeerName, b)
	//return len(b), nil

	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()

	//oc.GossipDM.SendToRemote(b)
	//if oc.RemoteAddresses.PeerName != math.MaxUint64 {
	if len(oc.RemoteAddresses) > 0 {
		peerNames := make([]mesh.PeerName, 0)
		oc.RemoteAddressesMtx.Lock()
		for _, remoteAddr := range oc.RemoteAddresses {
			peerNames = append(peerNames, remoteAddr.PeerName)
		}
		oc.RemoteAddressesMtx.Unlock()

		// when client side, length of peerNames is 1, so send data to only one stream
		// but when server side, length of peerNames is same as number of established streams
		// server side sends same data to all streams (it is ridiculous...)
		// because collect destination decidable design can't be implemented with mesh lib
		// stream collectly must work even if send data to not collect destination
		// by control of SCTP protocol layer...
		for _, peerName := range peerNames {
			oc.GossipDM.SendToRemote(peerName, oc.SessionSide, b)
		}
		//oc.GossipDM.SendToRemote(oc.RemoteAddresses.PeerName, oc.SessionSide, b)
	} else {
		// server side uses LastRecvPeer until Stream is established
		// because remote peer name can't be known until then
		oc.GossipDM.SendToRemote(oc.GossipDM.LastRecvPeer, oc.SessionSide, b)
	}

	return len(b), nil
}

// Close closes the conn and releases any Read calls
func (oc *GossipSession) Close() error {
	//return oc.pConn.WhenClose()
	oc.RemoteAddressesMtx.Lock()
	peerNames := make([]mesh.PeerName, 0)
	oc.RemoteAddressesMtx.Unlock()
	for _, peerName := range peerNames {
		oc.GossipDM.WhenClose(peerName)
	}
	oc.RemoteAddressesMtx.Unlock()

	//oc.GossipDM.WhenClose(oc.RemoteAddresses.PeerName)
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
	//if oc.RemoteAddresses != nil {
	//	return oc.RemoteAddresses
	//}
	oc.RemoteAddressesMtx.Lock()
	defer oc.RemoteAddressesMtx.Unlock()
	if len(oc.RemoteAddresses) > 0 {
		return oc.RemoteAddresses[0]
	}
	return nil
	//oc.SessMtx.RLock()
	//defer oc.SessMtx.RUnlock()
	//return oc.RemoteAddresses
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
