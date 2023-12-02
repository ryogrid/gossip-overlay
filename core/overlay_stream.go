package core

import (
	"fmt"
	"github.com/ryogrid/sctp"
	"github.com/weaveworks/mesh"
)

type OverlayStream struct {
	sctp.Stream
	localPeer      *Peer
	remotePeerName mesh.PeerName
	clientObj      *sctp.Association
	gossipSession  *GossipSession
	streamID       uint16
}

func NewOverlayStream(p *Peer, stream *sctp.Stream, clientObj *sctp.Association, gossipSession *GossipSession, streamID uint16) *OverlayStream {
	return &OverlayStream{
		Stream:         *stream,
		localPeer:      p,
		remotePeerName: gossipSession.RemoteAddress.PeerName,
		clientObj:      clientObj,
		gossipSession:  gossipSession,
		streamID:       streamID,
	}
}

// for resource release, need to call this method. other similar method call is not needed exept one of OverlayServer.
func (ols *OverlayStream) CloseOverlayStream() error {
	var err error = nil

	err1 := ols.Close()
	if err1 != nil {
		fmt.Println(err1)
		err = err1
	}
	err2 := ols.clientObj.Close()
	if err2 != nil {
		fmt.Println(err2)
		err = err2
	}
	err3 := ols.gossipSession.Close()
	if err3 != nil {
		fmt.Println(err3)
		err = err3
	}

	return err
}
