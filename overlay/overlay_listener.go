package overlay

import "net"

type OverlayListener struct {
}

func NewOverlayListener(ol *OverlayPeer) net.Listener {
	// TODO: need to implement (private_server.go::getGossipListener)
	return &OverlayListener{}
}

// Accept waits for and returns the next connection to the listener.
func (ol *OverlayListener) Accept() (net.Conn, error) {
	// TODO: need to implement (OverlayListerner::Accept)
	panic("not implemented")
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (ol *OverlayListener) Close() error {
	// TODO: need to implement (OverlayListerner::Close)
	panic("not implemented")
}

// Addr returns the listener's network address.
func (ol *OverlayListener) Addr() net.Addr {
	// TODO: need to implement (OverlayListerner::Addr)
	panic("not implemented")
}
