package overlay

import (
	"context"
	"github.com/pion/datachannel"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/gossip"
	"net"
	"time"
)

type OverlayStream struct {
	channel *datachannel.DataChannel
	oc      *OverlayClient
	assoc   *sctp.Association
	gsess   *gossip.GossipSession
}

func NewOverlayStream(channel *datachannel.DataChannel, oc *OverlayClient, assoc *sctp.Association, gsess *gossip.GossipSession) *OverlayStream {
	return &OverlayStream{channel, oc, assoc, gsess}
}

func (os *OverlayStream) LocalAddr() net.Addr {
	//TODO: Need to implement (OverlayStream.LocalAddr)
	panic("implement me")
}

func (os *OverlayStream) RemoteAddr() net.Addr {
	//TODO: Need to implement (OverlayStream.RemoteAddr)

	panic("implement me")
}

func (os *OverlayStream) SetDeadline(t time.Time) error {
	//TODO: Need to implement (OverlayStream.SetDeadline)
	panic("implement me")
}

func (os *OverlayStream) SetReadDeadline(t time.Time) error {
	//TODO: Need to implement (OverlayStream.SetReadDeadline)

	panic("implement me")
}

func (os *OverlayStream) SetWriteDeadline(t time.Time) error {
	//TODO: Need to implement (OverlayStream.SetWriteDeadline)
	panic("implement me")
}

func (os *OverlayStream) Read(p []byte) (n int, err error) {
	n, _, err = os.channel.ReadDataChannel(p)
	return n, err
}

func (os *OverlayStream) Write(p []byte) (n int, err error) {
	return os.channel.WriteDataChannel(p, false)
}

func (os *OverlayStream) Close() error {
	os.channel.Close()
	ctx := context.Background()
	return os.assoc.Shutdown(ctx)
}
