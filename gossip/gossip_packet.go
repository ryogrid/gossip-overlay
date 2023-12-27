package gossip

import (
	"bytes"
	"encoding/gob"
	"github.com/weaveworks/mesh"
)

type PacketKind uint8

const (
	PACKET_KIND_CTC_DATA PacketKind = iota
	PACKET_KIND_CTC_CONTROL
	PACKET_KIND_NOTIFY_PEER_INFO
)

type GossipPacket struct {
	FromPeer     mesh.PeerName
	ReceiverSide OperationSideAt
	StreamID     uint16
	SeqNum       uint64
	PktKind      PacketKind
	Buf          []byte
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

func decodeGossipPacket(buf []byte) (*GossipPacket, error) {
	var gp GossipPacket
	decBuf := bytes.NewBuffer(buf)
	if err := gob.NewDecoder(decBuf).Decode(&gp); err != nil {
		return nil, err
	}

	return &gp, nil
}
