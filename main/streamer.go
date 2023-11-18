package main

import (
	"flag"
	"fmt"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/ryogrid/gossip-overlay/core"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	peers := &util.Stringset{}
	var (
		side       = flag.String("side", "relay", "specify peer type (default: relay))")
		meshListen = flag.String("mesh", net.JoinHostPort("0.0.0.0", strconv.Itoa(mesh.Port)), "mesh listen address")
		hwaddr     = flag.String("hwaddr", util.MustHardwareAddr(), "MAC address, i.e. mesh Peer ID")
		nickname   = flag.String("nickname", util.MustHostname(), "Peer nickname")
		password   = flag.String("password", "", "password (optional)")
		channel    = flag.String("channel", "default", "gossip channel name")
		destname   = flag.String("destname", "", "destination Peer name (optional)")
	)
	flag.Var(peers, "peer", "initial Peer (may be repeated)")
	flag.Parse()

	logger := log.New(os.Stderr, *nickname+"> ", log.LstdFlags)

	host, portStr, err := net.SplitHostPort(*meshListen)
	if err != nil {
		logger.Fatalf("mesh address: %s: %v", *meshListen, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Fatalf("mesh address: %s: %v", *meshListen, err)
	}

	meshConf := mesh.Config{
		Host:               host,
		Port:               port,
		ProtocolMinVersion: mesh.ProtocolMaxVersion, //mesh.ProtocolMinVersion,
		Password:           []byte(*password),
		ConnLimit:          64,
		PeerDiscovery:      true,
		TrustedSubnets:     []*net.IPNet{},
	}

	name, err := mesh.PeerNameFromString(*hwaddr)
	if err != nil {
		logger.Fatalf("%s: %v", *hwaddr, err)
	}

	var destNameNum uint64 = math.MaxUint64
	if *destname != "" {
		destNameNum, err = strconv.ParseUint(*destname, 10, 64)
		if err != nil {
			logger.Fatalf("Could not parse Destname: %v", err)
		}
	}

	p := core.NewPeer(name, logger, mesh.PeerName(destNameNum), nickname, channel, meshListen, &meshConf, peers)

	defer func() {
		logger.Printf("mesh router stopping")
		p.Router.Stop()
	}()

	errs := make(chan error)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	if *side == "recv" {
		serverRoutine(p)
	} else if *side == "send" {
		clientRoutine(p)
	}

	logger.Print(<-errs)
}

func serverRoutine(p *core.Peer) {
	//conn, err := net.ListenUDP("udp", &addr)
	conn, err := p.GossipDataMan.NewGossipSessionForServer()
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	fmt.Println("created a udp listener")

	config := sctp.Config{
		//NetConn:       &disconnectedPacketConn{pConn: conn},
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err := sctp.Server(config)
	if err != nil {
		log.Panic(err)
	}
	defer a.Close()
	fmt.Println("created a server")

	stream, err := a.AcceptStream()
	if err != nil {
		log.Panic(err)
	}
	defer stream.Close()
	fmt.Println("accepted a stream")

	// set unordered = true and 10ms treshold for dropping packets
	//stream.SetReliabilityParams(true, sctp.ReliabilityTypeTimed, 10)
	stream.SetReliabilityParams(true, sctp.ReliabilityTypeReliable, 0)
	var pongSeqNum int
	for {
		buff := make([]byte, 1024)
		_, err = stream.Read(buff)
		if err != nil {
			log.Panic(err)
		}
		pingMsg := string(buff)
		fmt.Println("received:", pingMsg)

		fmt.Sscanf(pingMsg, "ping %d", &pongSeqNum)
		pongMsg := fmt.Sprintf("pong %d", pongSeqNum)
		_, err = stream.Write([]byte(pongMsg))
		if err != nil {
			log.Panic(err)
		}
		fmt.Println("sent:", pongMsg)

		time.Sleep(time.Second)
	}
}

func clientRoutine(p *core.Peer) {
	conn, err := p.GossipDataMan.NewGossipSessionForClient(p.Destname)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			panic(err)
		}
	}()
	fmt.Println("dialed udp ponger")

	config := sctp.Config{
		NetConn:       conn,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	}
	a, err := sctp.Client(config)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if closeErr := a.Close(); closeErr != nil {
			panic(err)
		}
	}()
	fmt.Println("created a client")

	stream, err := a.OpenStream(0, sctp.PayloadTypeWebRTCString)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			panic(err)
		}
	}()
	fmt.Println("opened a stream")

	// set unordered = true and 10ms treshold for dropping packets
	//stream.SetReliabilityParams(true, sctp.ReliabilityTypeTimed, 10)
	stream.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	go func() {
		var pingSeqNum int
		for {
			pingMsg := fmt.Sprintf("ping %d", pingSeqNum)
			_, err = stream.Write([]byte(pingMsg))
			if err != nil {
				log.Panic(err)
			}
			fmt.Println("sent:", pingMsg)

			pingSeqNum++

			time.Sleep(3 * time.Second)
		}
	}()

	for {
		buff := make([]byte, 1024)
		_, err = stream.Read(buff)
		if err != nil {
			log.Panic(err)
		}
		pongMsg := string(buff)
		fmt.Println("received:", pongMsg)
	}
}
