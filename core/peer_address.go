package core

import "github.com/weaveworks/mesh"

type PeerAddress struct {
	PeerName mesh.PeerName
}

func (na *PeerAddress) Network() string { // name of the network (for example, "tcp", "udp")
	return "gossip-overlay"
}

func (na *PeerAddress) String() string { // string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
	return na.PeerName.String()
}
