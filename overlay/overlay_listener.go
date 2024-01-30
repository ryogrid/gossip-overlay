package overlay

import (
	"fmt"
	"github.com/ryogrid/gossip-overlay/gossip"
	"net"
	"strconv"
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
// func (ol *OverlayListener) Accept() (net.Conn, error) {
func (ol *OverlayListener) Accept() (net.Conn, error) {
	fmt.Println("OverlayListener::Accept called", fmt.Sprintf("%d", ol.overlayServer.peer.GossipDataMan.Self))
	channel, _, _, _, err := ol.overlayServer.Accept()
	fmt.Println("OverlayListener::Accept fin", fmt.Sprintf("%v", err))
	return channel, err
	//
	//cert := "cert.pem" // e.g. '/path/to/my/client-cert.pem'
	//key := "key.pem"   // e.g. '/path/to/my/client-key.pem'

	//pool := x509.NewCertPool()
	//pem, err := ioutil.ReadFile(dbRootCert)
	//if err != nil {
	//	return nil, err
	//}
	//if ok := pool.AppendCertsFromPEM(pem); !ok {
	//	return nil, errors.New("unable to append root cert to pool")
	//}
	//certificate, err := tls.LoadX509KeyPair(cert, key)
	//if err != nil {
	//	panic(err)
	//	return nil, err
	//}
	//conn := tls.Server(channel, &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert, Certificates: []tls.Certificate{certificate}})
	//conn := tls.Server(channel, &tls.Config{ClientAuth: tls.NoClientCert, Certificates: []tls.Certificate{certificate}})
	//return conn, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (ol *OverlayListener) Close() error {
	// TODO: need to implement (OverlayListener::Close)
	// do nothing now

	ol.overlayServer.Close()
	fmt.Println("OverlayLister::Close called")
	return nil
}

// Addr returns the listener's network address.
func (ol *OverlayListener) Addr() net.Addr {
	fmt.Println("OverlayListener::Addr called")
	peerHost := ol.overlayServer.peer.Router.Host + ":" + strconv.Itoa(ol.overlayServer.peer.Router.Port)
	return &gossip.PeerAddress{
		PeerName: ol.overlayPeer.Peer.GossipDataMan.Self,
		PeerHost: &peerHost,
	}
}
