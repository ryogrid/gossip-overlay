package main

import (
	"flag"
	"fmt"
	"github.com/ryogrid/gossip-overlay/core"
	"github.com/ryogrid/gossip-overlay/overlay_setting"
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
		channel    = flag.String("channel", "default", "gossip channel name")
		destname   = flag.String("destname", "", "destination Peer name (optional)")
		debug      = flag.String("debug", "false", "print debug info, true of false (optional)")
	)
	flag.Var(peers, "peer", "initial Peer (may be repeated)")
	flag.Parse()

	if *debug == "true" {
		overlay_setting.OVERLAY_DEBUG = true
	}

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
		ProtocolMinVersion: mesh.ProtocolMaxVersion,
		Password:           nil,
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
	server, err := core.NewOverlayServer(p)
	if err != nil {
		panic(err)
	}

	stream, remotePeer, err := server.Accept()
	fmt.Println("stream accepted from ", remotePeer)
	if err != nil {
		panic(err)
	}

	var pongSeqNum = 100
	for {
		buff := make([]byte, 1024)
		util.OverlayDebugPrintln("before stream.Read")
		_, err = stream.Read(buff)
		util.OverlayDebugPrintln("after stream.Read", err, buff)
		if err != nil {
			//log.Panic(err)
			panic(err)
		}
		fmt.Println("received:", buff[0])

		sendBuf := []byte{byte(pongSeqNum % 255), buff[0]}
		_, err = stream.Write(sendBuf)
		if err != nil {
			//log.Panic(err)
			panic(err)
		}
		fmt.Println("sent:", sendBuf[0], sendBuf[1])
		pongSeqNum++

		time.Sleep(time.Second)
	}
}

func clientRoutine(p *core.Peer) {
	a, err := core.NewOverlayClient(p, p.Destname)
	if err != nil {
		panic(err)
	}

	stream, err2 := a.OpenStream()
	if err2 != nil {
		panic(err2)
	}

	go func() {
		var pingSeqNum int
		for {
			_, err = stream.Write([]byte{byte(pingSeqNum % 255)})
			if err != nil {
				//log.Panic(err)
				panic(err)
			}
			fmt.Println("sent:", pingSeqNum%255)

			pingSeqNum++

			time.Sleep(3 * time.Second)
		}
	}()

	for {
		buff := make([]byte, 1024)
		_, err = stream.Read(buff)
		if err != nil {
			//log.Panic(err)
			panic(err)
		}
		fmt.Println("received:", buff[0], buff[1])
	}
}
