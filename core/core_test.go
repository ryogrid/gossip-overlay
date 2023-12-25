package core

import (
	"fmt"
	"github.com/weaveworks/mesh"
	"testing"
)

func TestSerializeDeserialize(t *testing.T) {
	gp := GossipPacket{mesh.PeerName(0), []byte{1, 2, 3}, ClientSide, 1, 0, PACKET_KIND_CTC_DATA}
	//gp := GossipPacket{[]byte{}, ClientSide}

	buf := gp.Encode()
	fmt.Println(buf)

	gp2, err := DecodeGossipPacket(buf[0])
	if err != nil {
		panic(err)
	}
	fmt.Println(gp2)
	fmt.Println(gp2.Buf)
	fmt.Println(gp2.ReceiverSide)
}
