package core

import (
	"bytes"
	"encoding/gob"
	"github.com/weaveworks/mesh"
)

// type GossipPacket []byte
type GossipPacket struct {
	Buf          []byte
	ReceiverSide OperationSideAt
}

func (gp GossipPacket) Encode() [][]byte {
	var buf bytes.Buffer
	//if err := gob.NewEncoder(&buf2).Encode(buf); err != nil {
	if err := gob.NewEncoder(&buf).Encode(gp); err != nil {
		panic(err)
	}

	return [][]byte{buf.Bytes()}
}

func (gp GossipPacket) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	//retBuf := make([]byte, 0)
	//retBuf = append(retBuf, []byte(buf)...)
	//retBuf = append(retBuf, []byte(other.(GossipPacket))...)
	//
	//return GossipPacket(retBuf)

	retBuf := make([]byte, 0)
	retBuf = append(retBuf, gp.Buf...)
	retBuf = append(retBuf, other.(GossipPacket).Buf...)

	gp.Buf = retBuf
	return gp
}

func DecodeGossipPacket(buf []byte) (*GossipPacket, error) {
	var gp GossipPacket
	if err := gob.NewDecoder(bytes.NewReader(buf)).Decode(&gp); err != nil {
		return nil, err
	}

	return &gp, nil
}
