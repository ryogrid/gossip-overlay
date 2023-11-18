package core

import (
	"fmt"
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
	GossipDM *GossipDataManager
}

// GossipSetton implements net.Conn
var _ net.Conn = &GossipSession{}

// Read
func (oc *GossipSession) Read(b []byte) (n int, err error) {
	fmt.Println("GossipSession.Read called")
	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()
	buf := oc.GossipDM.Read(oc.RemoteAddress.PeerName)
	ret := make([]byte, len(buf))
	copy(ret, buf)
	b = ret

	return len(b), nil

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
	fmt.Println("GossipSession.Write called", b)

	////return oc.pConn.WriteTo(p, oc.RemoteAddr())
	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()
	//oc.GossipDM.Write(oc.RemoteAddress.PeerName, b)
	//return len(b), nil

	//oc.SessMtx.Lock()
	//defer oc.SessMtx.Unlock()
	oc.GossipDM.WriteToRemote(b)

	return len(b), nil
}

// Close closes the conn and releases any Read calls
func (oc *GossipSession) Close() error {
	//return oc.pConn.Close()
	oc.GossipDM.Close(oc.RemoteAddress.PeerName)
	return nil
}

func (oc *GossipSession) LocalAddr() net.Addr {
	fmt.Println("GossipSession.LocalAddr called", oc.LocalAddress.PeerName)
	if oc.LocalAddress != nil {
		return oc.LocalAddress
	}
	return nil
}

func (oc *GossipSession) RemoteAddr() net.Addr {
	fmt.Println("GossipSession.RemoteAddr called", oc.RemoteAddress.PeerName)
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
	fmt.Println("GossipSession.SetDeadline called", t)
	return nil
}

// SetReadDeadline is a stub
func (oc *GossipSession) SetReadDeadline(t time.Time) error {
	fmt.Println("GossipSession.SetReadDeadline called", t)
	return nil
}

// SetWriteDeadline is a stub
func (oc *GossipSession) SetWriteDeadline(t time.Time) error {
	fmt.Println("GossipSession.SetWriteDeadline called", t)
	return nil
}
