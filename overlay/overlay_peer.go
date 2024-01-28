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
)

var LoggerObj *log.Logger

type OverlayPeer struct {
	Peer *gossip.GossipPeer
}

func NewOverlayPeer(selfPeerId uint64, gossipListenPort int, peers *util.Stringset) (*OverlayPeer, error) {
	//name := mesh.PeerName(util.NewHashIDUint64(*host + ":" + strconv.Itoa(int(gossipListenPort))))
	//name := mesh.PeerName(util.NewHashIDUint16(*host + ":" + strconv.Itoa(int(gossipListenPort))))

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
	p := gossip.NewPeer(mesh.PeerName(selfPeerId), LoggerObj, &emptyStr, &emptyStr, &meshConf, peers)
	fmt.Println("NewOverlayPeer: peers=", peers.Slice())

	//remotePeerHost := meshConf.Host + ":" + strconv.Itoa(meshConf.Port)
	return &OverlayPeer{p}, nil
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
