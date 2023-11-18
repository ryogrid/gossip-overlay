package core

import (
	"net"
	"sync"
	"time"
)

// represents a gossip session throuth remote Peer
type GossipSession struct {
	RemoteAddress *PeerAddress
	SessMtx       sync.RWMutex
	// buffered data is accessed through GossipDM each time
	GossipDM *GossipDataManager
}

// Read
func (oc *GossipSession) Read(p []byte) (int, error) {
	i, rAddr, err := oc.pConn.ReadFrom(p)
	if err != nil {
		return 0, err
	}

	oc.SessMtx.Lock()
	oc.rAddr = rAddr
	oc.SessMtx.Unlock()

	return i, err
}

// Write writes len(p) bytes from p to the DTLS connection
func (oc *GossipSession) Write(p []byte) (n int, err error) {
	return oc.pConn.WriteTo(p, oc.RemoteAddr())
}

// Close closes the conn and releases any Read calls
func (oc *GossipSession) Close() error {
	return oc.pConn.Close()
}

// LocalAddr is a stub
func (oc *GossipSession) LocalAddr() net.Addr {
	if oc.pConn != nil {
		return oc.pConn.LocalAddr()
	}
	return nil
}

// RemoteAddress is a stub
func (oc *GossipSession) RemoteAddr() net.Addr {
	oc.SessMtx.RLock()
	defer oc.SessMtx.RUnlock()
	return oc.RemoteAddress
}

// SetDeadline is a stub
func (oc *GossipSession) SetDeadline(time.Time) error {
	return nil
}

// SetReadDeadline is a stub
func (oc *GossipSession) SetReadDeadline(time.Time) error {
	return nil
}

// SetWriteDeadline is a stub
func (oc *GossipSession) SetWriteDeadline(time.Time) error {
	return nil
}
