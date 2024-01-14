package gossip

import (
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"io/ioutil"
	"log"
)

// GossipPeer encapsulates GossipDataManager, GossipMessageManager and implements mesh.Gossiper.
// It should be passed to mesh.Router.NewGossip,
// and the resulting Gossip registered in turn,
// before calling mesh.Router.Start.
type GossipPeer struct {
	GossipDataMan *GossipDataManager
	GossipMM      *GossipMessageManager
	send          mesh.Gossip
	actions       chan<- func()
	quit          chan struct{}
	logger        *log.Logger
	//Destname      mesh.PeerName
	Router *mesh.Router
}

// GossipPeer implements mesh.Gossiper.
var _ mesh.Gossiper = &GossipPeer{}

// Construct a GossipPeer with empty GossipDataManager.
// Be sure to registerGossipObj a channel, later,
// so we can make outbound communication.
func NewPeer(self mesh.PeerName, logger *log.Logger, nickname *string, channel *string, meshConf *mesh.Config, peers *util.Stringset) *GossipPeer {
	router, err := mesh.NewRouter(*meshConf, self, *nickname, mesh.NullOverlay{}, log.New(ioutil.Discard, "", 0))

	if err != nil {
		logger.Fatalf("Could not create router: %v", err)
	}

	actions := make(chan func())
	tmpDM := NewGossipDataManager(self)
	p := &GossipPeer{
		GossipDataMan: tmpDM,
		GossipMM:      NewGossipMessageManager(&PeerAddress{self}, tmpDM),
		send:          nil, // must .registerGossipObj() later
		actions:       actions,
		quit:          make(chan struct{}),
		logger:        logger,
		//Destname:      destname,
		Router: router,
	}
	p.GossipDataMan.peer = p

	go p.loop(actions)

	gossip, err := router.NewGossip(*channel, p)
	if err != nil {
		logger.Fatalf("Could not create gossip: %v", err)
	}

	p.registerGossipObj(gossip)

	go func() {
		logger.Printf("mesh router starting (%d)", self)
		router.Start()
	}()

	router.ConnectionMaker.InitiateConnections(peers.Slice(), true)
	return p
}

func (p *GossipPeer) loop(actions <-chan func()) {
	for {
		select {
		case f := <-actions:
			f()
		case <-p.quit:
			return
		}
	}
}

// registerGossipObj the result of a mesh.Router.NewGossip.
func (p *GossipPeer) registerGossipObj(send mesh.Gossip) {
	p.actions <- func() { p.send = send }
}

func (p *GossipPeer) Stop() {
	close(p.quit)
}

// Return a copy of our complete GossipDataManager.
func (p *GossipPeer) Gossip() (complete mesh.GossipData) {
	util.OverlayDebugPrintln("Gossip called")
	return GossipPacket{}
}

// Merge the gossiped data represented by Buf into our GossipDataManager.
// Return the GossipDataManager information that was modified.
func (p *GossipPeer) OnGossip(buf []byte) (delta mesh.GossipData, err error) {
	util.OverlayDebugPrintln("OnGossip called")
	return GossipPacket{}, nil
}

// Merge the gossiped data represented by Buf into our GossipDataManager.
// Return the GossipDataManager information that was modified.
func (p *GossipPeer) OnGossipBroadcast(src mesh.PeerName, buf []byte) (received mesh.GossipData, err error) {
	panic("OnGossipBroadcast can not be called now")
}

// Merge the gossiped data represented by Buf into our GossipDataManager.
func (p *GossipPeer) OnGossipUnicast(src mesh.PeerName, buf []byte) error {
	util.OverlayDebugPrintln("OnGossipUnicast called")
	return p.GossipMM.onPacketReceived(src, buf)
}
