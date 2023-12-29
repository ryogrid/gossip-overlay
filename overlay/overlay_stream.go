package overlay

import (
	"context"
	"github.com/pion/datachannel"
	"github.com/pion/sctp"
)

type OverlayStream struct {
	channel *datachannel.DataChannel
	assoc   *sctp.Association
}

func NewOverlayStream(channel *datachannel.DataChannel, assoc *sctp.Association) *OverlayStream {
	return &OverlayStream{channel, assoc}
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
