package overlay

import (
	"fmt"
	"github.com/ryogrid/gossip-overlay/gossip"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"log"
	"math"
	"net"
	"os"
	"strconv"
)

var LoggerObj *log.Logger

type OverlayPeer struct {
	Peer *gossip.GossipPeer
	// when ussing OverlayListener, this field is not used
	olServ *OverlayServer
}

func NewOverlayPeer(selfPeerId uint64, gossipListenHost *string, gossipListenPort int, peers *util.Stringset, isUseOnProxy bool) (*OverlayPeer, error) {
	meshConf := mesh.Config{
		Host:               "0.0.0.0",
		Port:               gossipListenPort,
		ProtocolMinVersion: mesh.ProtocolMaxVersion,
		Password:           nil,
		ConnLimit:          64,
		PeerDiscovery:      true,
		TrustedSubnets:     []*net.IPNet{},
	}

	LoggerObj = log.New(os.Stderr, "gossip> ", log.LstdFlags)
	emptyStr := ""

	var peerHostAndPort string
	peerHostAndPort = *gossipListenHost + ":" + strconv.Itoa(gossipListenPort)
	if isUseOnProxy {
		// proxy's host view on application layer should match proxied application working address
		// (convention: proxy is launched at proxied application working port + 2)
		peerHostAndPort = *gossipListenHost + ":" + strconv.Itoa(gossipListenPort-2)
	}

	p := gossip.NewPeer(mesh.PeerName(selfPeerId), &peerHostAndPort, LoggerObj, &emptyStr, &emptyStr, &meshConf, peers)
	fmt.Println("NewOverlayPeer: peers=", peers.Slice())

	//remotePeerHost := meshConf.Host + ":" + strconv.Itoa(meshConf.Port)
	return &OverlayPeer{p, nil}, nil
}

func (olPeer *OverlayPeer) OpenStreamToTargetPeer(peerId mesh.PeerName, remotePeerHost string) net.Conn {
	LoggerObj.Println(fmt.Sprintf("Opening a stream to %d", peerId))

	oc, err := NewOverlayClient(olPeer.Peer, peerId, remotePeerHost, olPeer.Peer.GossipMM)
	if err != nil {
		panic(err)
	}

	channel, streamID, err2 := oc.OpenChannel(math.MaxUint16)
	if err2 != nil {
		return nil
	}
	fmt.Println(fmt.Sprintf("opened: %d", streamID))

	return channel
}

func (olPeer *OverlayPeer) GetOverlayListener() net.Listener {
	return NewOverlayListener(olPeer)
}

func (olPeer *OverlayPeer) Accept() (stream *OverlayStream, remotePeerId mesh.PeerName, remotePeerHost *string, streamId uint16, err error) {
	if olPeer.olServ == nil {
		oserv, err_ := NewOverlayServer(olPeer.Peer, olPeer.Peer.GossipMM)
		if err_ != nil {
			panic(err_)
		}
		olPeer.olServ = oserv
	}
	return olPeer.olServ.Accept()
}
