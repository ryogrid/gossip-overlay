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
	"runtime"
	"strconv"
	"syscall"
	"time"
)

/*
.\streamer.exe -side recv -hwaddr 00:00:00:00:00:02 -nickname b -mesh :6002 -debug false | Tee-Object -FilePath ".\recv-1.txt"
.\streamer.exe -side send -hwaddr 00:00:00:00:00:03 -nickname c -mesh :6003 -destname 2 -peer 127.0.0.1:6002 -debug false | Tee-Object -FilePath ".\send-1.txt"
*/
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
		logger.Fatalf("mesh host: %s: %v", *meshListen, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		logger.Fatalf("mesh port: %d: %v", port, err)
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

	runtime.GOMAXPROCS(10)
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

	if *side == "send" {
		go clientRoutine(p)
	} else if *side == "recv" {
		go serverRoutine(p)
	} else {
		//panic("invalid side")
		// do nothing (relay)
	}

	logger.Print(<-errs)
}

func serverRoutine(p *core.Peer) {
	util.OverlayDebugPrintln("start serverRoutine")
	oserv, err := core.NewOverlayServer(p, p.GossipMM)
	if err != nil {
		panic(err)
	}

	channel, remotePeerName, streamID, err2 := oserv.Accept()
	if err2 != nil {
		panic(err2)
	}
	fmt.Println("accepted:", remotePeerName, streamID)

	pongSeqNum := 0
	for {
		util.OverlayDebugPrintln("call ReadDataChannel!")
		buff := make([]byte, 1024)
		n1, _, err3 := channel.ReadDataChannel(buff)
		if err3 != nil || n1 != 1 {
			util.OverlayDebugPrintln("panic occured at ReadDataChannel!", err3, n1)
			panic(err)
		}
		fmt.Println("received:", buff[0])

		util.OverlayDebugPrintln("call WriteDataChannel!")
		n2, err4 := channel.WriteDataChannel([]byte{byte(pongSeqNum % 255), buff[0]}, false)
		if err4 != nil || n2 != 2 {
			panic(err4)
		}
		fmt.Println("sent:", pongSeqNum%255, buff[0])
		pongSeqNum++
	}
}

func clientRoutine(p *core.Peer) {
	util.OverlayDebugPrintln("start clientRoutine")
	oc, err := core.NewOverlayClient(p, p.Destname, p.GossipMM)
	if err != nil {
		panic(err)
	}

	channel, streamID, err2 := oc.OpenChannel(math.MaxUint16)
	if err2 != nil {
		panic(err2)
	}
	fmt.Println("opened:", streamID)

	pingSeqNum := 0
	for {
		util.OverlayDebugPrintln("call WriteDataChannel!")
		n, err3 := channel.WriteDataChannel([]byte{byte(pingSeqNum % 255)}, false)
		if err3 != nil || n != 1 {
			util.OverlayDebugPrintln("panic occured at WriteDataChannel!", err3, n)
			panic(err3)
		}
		fmt.Println("sent:", pingSeqNum%255)
		pingSeqNum++

		util.OverlayDebugPrintln("call ReadDataChannel!")
		buff := make([]byte, 1024)
		n, _, err = channel.ReadDataChannel(buff)
		if err != nil || n != 2 {
			util.OverlayDebugPrintln("panic occured at ReadDataChannel!", err, n)
			panic(err)
		}
		fmt.Println("received:", buff[0], buff[1])

		time.Sleep(3 * time.Second)
	}
}
