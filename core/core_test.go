package core

import (
	"fmt"
	"testing"
)

func TestSerializeDeserialize(t *testing.T) {
	gp := GossipPacket{[]byte{1, 2, 3}, ClientSide}
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
