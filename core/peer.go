package core

import (
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"io/ioutil"
	"log"
)

// Peer encapsulates GossipDataManager, GossipMessageManager and implements mesh.Gossiper.
// It should be passed to mesh.Router.NewGossip,
// and the resulting Gossip registered in turn,
// before calling mesh.Router.Start.
type Peer struct {
	GossipDataMan *GossipDataManager
	GossipMM      *GossipMessageManager
	Send          mesh.Gossip
	Actions       chan<- func()
	Quit          chan struct{}
	Logger        *log.Logger
	Destname      mesh.PeerName
	Router        *mesh.Router
}

// Peer implements mesh.Gossiper.
var _ mesh.Gossiper = &Peer{}

// Construct a Peer with empty GossipDataManager.
// Be sure to RegisterGossipObj a channel, later,
// so we can make outbound communication.
func NewPeer(self mesh.PeerName, logger *log.Logger, destname mesh.PeerName, nickname *string, channel *string, meshListen *string, meshConf *mesh.Config, peers *util.Stringset) *Peer {
	router, err := mesh.NewRouter(*meshConf, self, *nickname, mesh.NullOverlay{}, log.New(ioutil.Discard, "", 0))

	if err != nil {
		logger.Fatalf("Could not create router: %v", err)
	}

	actions := make(chan func())
	tmpDM := NewGossipDataManager(self)
	p := &Peer{
		GossipDataMan: tmpDM,
		GossipMM:      NewGossipMessageManager(&PeerAddress{self}, tmpDM),
		Send:          nil, // must .RegisterGossipObj() later
		Actions:       actions,
		Quit:          make(chan struct{}),
		Logger:        logger,
		Destname:      destname,
		Router:        router,
	}
	p.GossipDataMan.Peer = p

	go p.loop(actions)

	gossip, err := router.NewGossip(*channel, p)
	if err != nil {
		logger.Fatalf("Could not create gossip: %v", err)
	}

	p.RegisterGossipObj(gossip)

	go func() {
		logger.Printf("mesh router starting (%s)", *meshListen)
		router.Start()
	}()

	router.ConnectionMaker.InitiateConnections(peers.Slice(), true)
	return p
}

func (p *Peer) loop(actions <-chan func()) {
	for {
		select {
		case f := <-actions:
			f()
		case <-p.Quit:
			return
		}
	}
}

// RegisterGossipObj the result of a mesh.Router.NewGossip.
func (p *Peer) RegisterGossipObj(send mesh.Gossip) {
	p.Actions <- func() { p.Send = send }
}

func (p *Peer) Stop() {
	close(p.Quit)
}

// Return a copy of our complete GossipDataManager.
func (p *Peer) Gossip() (complete mesh.GossipData) {
	util.OverlayDebugPrintln("Gossip called")
	return GossipPacket{}
}

// Merge the gossiped data represented by buf into our GossipDataManager.
// Return the GossipDataManager information that was modified.
func (p *Peer) OnGossip(buf []byte) (delta mesh.GossipData, err error) {
	util.OverlayDebugPrintln("OnGossip called")
	return GossipPacket{}, nil
}

// Merge the gossiped data represented by buf into our GossipDataManager.
// Return the GossipDataManager information that was modified.
func (p *Peer) OnGossipBroadcast(src mesh.PeerName, buf []byte) (received mesh.GossipData, err error) {
	panic("OnGossipBroadcast can not be called now")
}

// Merge the gossiped data represented by buf into our GossipDataManager.
func (p *Peer) OnGossipUnicast(src mesh.PeerName, buf []byte) error {
	util.OverlayDebugPrintln("OnGossipUnicast called")
	return p.GossipMM.OnPacketReceived(src, buf)
}
