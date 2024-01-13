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
}

func NewOverlayPeer(host *string, gossipListenPort uint16, peers *util.Stringset) (*OverlayPeer, error) {
	name := mesh.PeerName(util.NewHashIDUint64(*host + ":" + strconv.Itoa(int(gossipListenPort))))

	meshConf := mesh.Config{
		Host:               "0.0.0.0",
		Port:               int(gossipListenPort),
		ProtocolMinVersion: mesh.ProtocolMaxVersion,
		Password:           nil,
		ConnLimit:          64,
		PeerDiscovery:      true,
		TrustedSubnets:     []*net.IPNet{},
	}

	LoggerObj = log.New(os.Stderr, "gossip> ", log.LstdFlags)
	emptyStr := ""
	meshListen := "local"
	p := gossip.NewPeer(name, LoggerObj, &emptyStr, &emptyStr, &meshListen, &meshConf, peers)

	return &OverlayPeer{p}, nil
}

func (olPeer *OverlayPeer) OpenStreamToTargetPeer(peerId mesh.PeerName) net.Conn {
	LoggerObj.Println(fmt.Sprintf("Opening a stream to %d", peerId))

	oc, err := NewOverlayClient(olPeer.Peer, peerId, olPeer.Peer.GossipMM)
	if err != nil {
		panic(err)
	}

	channel, streamID, err2 := oc.OpenChannel(math.MaxUint16)
	if err2 != nil {
		panic(err2)
	}
	fmt.Println(fmt.Sprintf("opened: %d", streamID))

	return channel
}

func (olPeer *OverlayPeer) GetOverlayListener() net.Listener {
	return NewOverlayListener(olPeer)
}
