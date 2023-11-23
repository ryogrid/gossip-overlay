package core

import (
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"io/ioutil"
	"log"
)

type PeerType int

// TODO: temporal definition
const (
	Relay PeerType = iota
	Server
	Client
)

// Peer encapsulates GossipDataManager and implements mesh.Gossiper.
// It should be passed to mesh.Router.NewGossip,
// and the resulting Gossip registered in turn,
// before calling mesh.Router.Start.
type Peer struct {
	GossipDataMan *GossipDataManager
	Send          mesh.Gossip
	Actions       chan<- func()
	Quit          chan struct{}
	Logger        *log.Logger
	Destname      mesh.PeerName
	Router        *mesh.Router
	Type          PeerType
}

// Peer implements mesh.Gossiper.
var _ mesh.Gossiper = &Peer{}

// Construct a Peer with empty GossipDataManager.
// Be sure to Register a channel, later,
// so we can make outbound communication.
func NewPeer(self mesh.PeerName, logger *log.Logger, destname mesh.PeerName, nickname *string, channel *string, meshListen *string, meshConf *mesh.Config, peers *util.Stringset) *Peer {
	router, err := mesh.NewRouter(*meshConf, self, *nickname, mesh.NullOverlay{}, log.New(ioutil.Discard, "", 0))

	if err != nil {
		logger.Fatalf("Could not create router: %v", err)
	}

	actions := make(chan func())
	p := &Peer{
		GossipDataMan: NewGossipDataManager(self),
		Send:          nil, // must .Register() later
		Actions:       actions,
		Quit:          make(chan struct{}),
		Logger:        logger,
		Destname:      destname,
		Router:        router,
		Type:          -1, // must set later
	}
	p.GossipDataMan.Peer = p

	go p.loop(actions)

	gossip, err := router.NewGossip(*channel, p)
	if err != nil {
		logger.Fatalf("Could not create gossip: %v", err)
	}

	p.Register(gossip)

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

// Register the result of a mesh.Router.NewGossip.
func (p *Peer) Register(send mesh.Gossip) {
	p.Actions <- func() { p.Send = send }
}

//func (p *Peer) ReadPeer(fromPeer mesh.PeerName) []byte {
//	return p.GossipDataMan.Read(fromPeer)
//}
//
//func (p *Peer) WritePeer() (result []byte) {
//	c := make(chan struct{})
//	p.Actions <- func() {
//		defer close(c)
//		val1 := byte(rand.Int31() % 256)
//		val2 := byte(rand.Int31() % 256)
//		//GossipDM := p.GossipDM.WritePeer([]byte{val1, val2})
//		sendData := []byte{val1, val2}
//		if p.Send != nil {
//			//p.Send.GossipBroadcast(GossipDM)
//			util.OverlayDebugPrintln("WritePeer", sendData)
//			p.Send.GossipUnicast(p.Destname, sendData)
//		} else {
//			p.Logger.Printf("no sender configured; not broadcasting update right now")
//		}
//		result = []byte{}
//	}
//	<-c
//	return result
//}

func (p *Peer) Stop() {
	close(p.Quit)
}

// Return a copy of our complete GossipDataManager.
func (p *Peer) Gossip() (complete mesh.GossipData) {
	util.OverlayDebugPrintln("Gossip called")
	//return GossipPacket([]byte{})
	return GossipPacket{}
}

// Merge the gossiped data represented by buf into our GossipDataManager.
// Return the GossipDataManager information that was modified.
func (p *Peer) OnGossip(buf []byte) (delta mesh.GossipData, err error) {
	util.OverlayDebugPrintln("OnGossip called")
	//return GossipPacket(buf), nil
	return GossipPacket{}, nil
}

// Merge the gossiped data represented by buf into our GossipDataManager.
// Return the GossipDataManager information that was modified.
func (p *Peer) OnGossipBroadcast(src mesh.PeerName, buf []byte) (received mesh.GossipData, err error) {
	panic("OnGossipBroadcast can not be called now")
	//util.OverlayDebugPrintln("OnGossipBroadcast called")
	//var data []byte
	//if err1 := gob.NewDecoder(bytes.NewReader(buf)).Decode(&data); err != nil {
	//	return nil, err1
	//}
	//
	////received = p.GossipDataMan.MergeReceived(p, src, data)
	//received = p.GossipDataMan.MergeComplete(p, src, data)
	//if received == nil {
	//	p.Logger.Printf("OnGossipBroadcast %s %v => delta %v", src, data, received)
	//} else {
	//	p.Logger.Printf("OnGossipBroadcast %s %v => delta %v", src, data, received.(*GossipDataManager).bufs)
	//}
	//return received, nil
}

// Merge the gossiped data represented by buf into our GossipDataManager.
func (p *Peer) OnGossipUnicast(src mesh.PeerName, buf []byte) error {
	util.OverlayDebugPrintln("OnGossipUnicast called")
	// decoding is not needed when GossipDataManager is []byte

	//if err := gob.NewDecoder(bytes.NewReader(buf)).Decode(&set); err != nil {
	//	return nil, err
	//}
	gp, err := DecodeGossipPacket(buf)
	if err != nil {
		panic(err)
	}

	//err2 := p.GossipDataMan.WriteToLocalBuffer(p, src, buf)
	err2 := p.GossipDataMan.WriteToLocalBuffer(p, src, gp.ReceiverSide, gp.Buf)
	if err2 != nil {
		panic(err2)
	}
	p.Logger.Printf("OnGossipUnicast %s %v", src, buf)
	return nil
}
