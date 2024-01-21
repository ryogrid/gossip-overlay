package overlay

import (
	"context"
	"fmt"
	"github.com/pion/datachannel"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/gossip"
	"net"
	"sync"
	"time"
)

type OverlayStream struct {
	channel     *datachannel.DataChannel
	oc          *OverlayClient
	assoc       *sctp.Association
	gsess       *gossip.GossipSession
	localBuf    []byte
	localBufMtx *sync.Mutex
}

func NewOverlayStream(channel *datachannel.DataChannel, oc *OverlayClient, assoc *sctp.Association, gsess *gossip.GossipSession) *OverlayStream {
	return &OverlayStream{channel, oc, assoc, gsess, make([]byte, 0), new(sync.Mutex)}
}

func (os *OverlayStream) LocalAddr() net.Addr {
	return os.gsess.LocalAddr()
	//return &gossip.PeerAddress{
	//	PeerName: os.oc.peer.GossipDataMan.Self,,
	//}
}

func (os *OverlayStream) RemoteAddr() net.Addr {
	fmt.Println("OverlayStream::RemoteAddr called ", os.gsess.RemoteAddr().String())
	return os.gsess.RemoteAddr()
	//return &gossip.PeerAddress{
	//	os.oc.remotePeerName,
	//}
}

func (os *OverlayStream) SetDeadline(t time.Time) error {
	os.channel.SetReadDeadline(t)
	return nil
}

func (os *OverlayStream) SetReadDeadline(t time.Time) error {
	os.channel.SetReadDeadline(t)
	return nil
}

func (os *OverlayStream) SetWriteDeadline(t time.Time) error {
	os.channel.SetReadDeadline(t)
	return nil
}

func (os *OverlayStream) Read(p []byte) (n int, err error) {
	//os.localBufMtx.Lock()
	//tmpBuf := make([]byte, 0)
	//n, _, err = os.channel.ReadDataChannel(tmpBuf)
	//os.localBuf = append(os.localBuf, tmpBuf...)
	//retBuf := make([]byte, 0)
	//if len(os.localBuf) > MaxPayloadSizeOnOverlayStream {
	//	copy(retBuf, os.localBuf[:MaxPayloadSizeOnOverlayStream])
	//	os.localBuf = os.localBuf[MaxPayloadSizeOnOverlayStream:]
	//} else {
	//	copy(retBuf, os.localBuf)
	//	os.localBuf = make([]byte, 0)
	//}
	//p = retBuf
	//os.localBufMtx.Unlock()
	fmt.Println("OverlayStream::Read called cap(p)=", cap(p))
	n_, _, err_ := os.channel.ReadDataChannel(p)
	if n == 0 {
		fmt.Printf("OverlayStream::Read: n == 0, %v", err)
	}
	fmt.Println("OverlayStream::Read called", n_, err)
	return n_, err_
}

func (os *OverlayStream) Write(p []byte) (n int, err error) {
	n_, err_ := os.channel.WriteDataChannel(p, false)
	fmt.Println("OverlayStream::Write called", n_, err_)
	return n_, err_
}

func (os *OverlayStream) Close() error {
	os.channel.Close()
	ctx := context.Background()
	os.oc.Destroy()
	return os.assoc.Shutdown(ctx)
}
