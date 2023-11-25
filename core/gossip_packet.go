package core

import (
	"bytes"
	"encoding/gob"
	"github.com/weaveworks/mesh"
)

type GossipPacket struct {
	Buf          []byte
	ReceiverSide OperationSideAt
}

func (gp GossipPacket) Encode() [][]byte {
	buf := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(buf).Encode(&gp); err != nil {
		panic(err)
	}

	return [][]byte{buf.Bytes()}
}

func (gp GossipPacket) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	retBuf := make([]byte, 0)
	retBuf = append(retBuf, gp.Buf...)
	retBuf = append(retBuf, other.(GossipPacket).Buf...)

	gp.Buf = retBuf
	return gp
}

func DecodeGossipPacket(buf []byte) (*GossipPacket, error) {
	var gp GossipPacket
	decBuf := bytes.NewBuffer(buf)
	if err := gob.NewDecoder(decBuf).Decode(&gp); err != nil {
		return nil, err
	}

	return &gp, nil
}
