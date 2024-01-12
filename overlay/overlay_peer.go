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
	"time"
)

var LoggerObj *log.Logger

type OverlayPeer struct {
	Peer *gossip.GossipPeer
}

func NewNode(destPeerId *uint64, gossipListenPort uint16) (*OverlayPeer, error) {
	//nicAddr := util.MustHardwareAddr()
	//name, err := mesh.PeerNameFromString(nicAddr)
	//if err != nil {
	//	panic("Failed to get PeerName from NIC address")
	//}
	name := mesh.PeerName(time.Now().Unix())

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
	var destPeerId_ uint64 = math.MaxUint64
	if destPeerId != nil {
		destPeerId_ = *destPeerId
	}
	peers := &util.Stringset{}
	peers.Set(constants.BootstrapPeer)
	p := gossip.NewPeer(name, LoggerObj, mesh.PeerName(destPeerId_), &emptyStr, &emptyStr, &meshListen, &meshConf, peers)

	return &OverlayPeer{p}, nil
}

func (node *OverlayPeer) OpenStreamToTargetPeer(peerId mesh.PeerName) net.Conn {
	LoggerObj.Println(fmt.Sprintf("Opening a stream to %d", peerId))

	oc, err := NewOverlayClient(node.Peer, peerId, node.Peer.GossipMM)
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
