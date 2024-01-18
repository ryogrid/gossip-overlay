package gossip

import "github.com/weaveworks/mesh"

type PeerAddress struct {
	PeerName mesh.PeerName
	PeerHost *string
}

func (na *PeerAddress) Network() string { // name of the network (for example, "tcp", "udp")
	//return "gossip-overlay"
	return "tcp"
}

func (na *PeerAddress) String() string {
	//return na.PeerName.String()
	if na.PeerHost == nil {
		return "127.0.0.1"
	}
	return *na.PeerHost
}
