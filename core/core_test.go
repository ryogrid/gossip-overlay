package core

import (
	"fmt"
	"github.com/ryogrid/gossip-overlay/gossip"
	"github.com/weaveworks/mesh"
	"testing"
)

func TestSerializeDeserialize(t *testing.T) {
	gp := gossip.GossipPacket{mesh.PeerName(0), []byte{1, 2, 3}, gossip.ClientSide, 1, 0, gossip.PACKET_KIND_CTC_DATA}
	//gp := GossipPacket{[]byte{}, ClientSide}

	buf := gp.Encode()
	fmt.Println(buf)

	gp2, err := gossip.DecodeGossipPacket(buf[0])
	if err != nil {
		panic(err)
	}
	fmt.Println(gp2)
	fmt.Println(gp2.Buf)
	fmt.Println(gp2.ReceiverSide)
}
