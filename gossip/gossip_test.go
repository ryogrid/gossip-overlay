package gossip

import (
	"fmt"
	"github.com/weaveworks/mesh"
	"testing"
)

func TestSerializeDeserialize(t *testing.T) {
	gp := GossipPacket{mesh.PeerName(0), "127.0.0.1", ClientSide, 1, 0, PACKET_KIND_CTC_DATA, []byte{1, 2, 3}}
	//gp := GossipPacket{[]byte{}, ClientSide}

	buf := gp.Encode()
	fmt.Println(buf)

	gp2, err := decodeGossipPacket(buf[0])
	if err != nil {
		panic(err)
	}
	fmt.Println(gp2)
	fmt.Println(gp2.Buf)
	fmt.Println(gp2.ReceiverSide)
}
