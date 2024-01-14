package overlay

import (
	"fmt"
	"github.com/ryogrid/gossip-overlay/gossip"
	"net"
)

type OverlayListener struct {
	overlayPeer   *OverlayPeer
	overlayServer *OverlayServer
}

func NewOverlayListener(ol *OverlayPeer) net.Listener {
	oserv, err := NewOverlayServer(ol.Peer, ol.Peer.GossipMM)
	if err != nil {
		panic(err)
	}

	return &OverlayListener{
		overlayPeer:   ol,
		overlayServer: oserv,
	}
}

// Accept waits for and returns the next connection to the listener.
func (ol *OverlayListener) Accept() (net.Conn, error) {
	fmt.Println("OverlayListener::Accept called", fmt.Sprintf("%d", ol.overlayServer.peer.GossipDataMan.Self))
	channel, _, _, err := ol.overlayServer.Accept()
	fmt.Println("OverlayListener::Accept fin", fmt.Sprintf("%v", err))
	return channel, err
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (ol *OverlayListener) Close() error {
	// TODO: need to implement (OverlayListener::Close)
	// do nothing now

	fmt.Println("OverlayLister::Close called")
	return nil
}

// Addr returns the listener's network address.
func (ol *OverlayListener) Addr() net.Addr {
	fmt.Println("OverlayListener::Addr called")
	return &gossip.PeerAddress{
		PeerName: ol.overlayPeer.Peer.GossipDataMan.Self,
	}
}
