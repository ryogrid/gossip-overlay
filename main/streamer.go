package main

import (
	"context"
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
		streamID   = flag.String("streamid", "", "stream ID (optional)")
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
		//go serverRoutine(p)
	} else if *side == "send" {
		convedStreamID, err2 := strconv.ParseUint(*streamID, 10, 16)
		if err2 != nil {
			panic(err2)
		}
		fmt.Println("convedStreamID:", convedStreamID)
		go clientRoutine(p, uint16(convedStreamID))
	}

	logger.Print(<-errs)
}

//func serverRoutine(p *core.Peer) {
//	server, err := core.NewOverlayServer(p)
//	if err != nil {
//		panic(err)
//	}
//
//	for {
//		overlayStream, remotePeer, err2 := server.Accept()
//		fmt.Println("overlay overlayStream accepted from ", remotePeer)
//		if err2 != nil {
//			panic(err2)
//		}
//
//		go func(stream_ *core.OverlayStream) {
//			var pongSeqNum = 100
//			for {
//				buff := make([]byte, 1024)
//				util.OverlayDebugPrintln("before overlayStream.Read")
//				_, err = overlayStream.Read(buff)
//				util.OverlayDebugPrintln("after overlayStream.Read", err, buff)
//				if err != nil {
//					panic(err)
//				}
//				fmt.Println("received:", buff[0])
//
//				sendBuf := []byte{byte(pongSeqNum % 255), buff[0]}
//				_, err = overlayStream.Write(sendBuf)
//				if err != nil {
//					panic(err)
//				}
//				fmt.Println("sent:", sendBuf[0], sendBuf[1])
//				pongSeqNum++
//
//				time.Sleep(time.Second)
//			}
//		}(overlayStream)
//	}
//}

func clientRoutine(p *core.Peer, streamId uint16) {
	util.OverlayDebugPrintln("start clientRoutine")
	a, err := core.NewOverlayClient(p, p.Destname)
	if err != nil {
		panic(err)
	}

	stream1, stream2, a2_2, err2 := a.OpenStream(streamId)
	if err2 != nil {
		panic(err2)
	}

	recvedByte := byte(0)

	// TODO: temporal impl
	if p.GossipDataMan.Self == 1 { //  reader
		for {
			buff := make([]byte, 1024)
			_, _, err = stream1.ReadDataChannel(buff)
			if err != nil {
				//log.Panic(err)
				panic(err)
			}
			fmt.Println("received:", buff[0], buff[1])
		}
	} else if p.GossipDataMan.Self == 2 { // writer and reader
		var pingSeqNum int
		for {
			//_, err = stream.WriteSCTP([]byte{byte(pingSeqNum % 255), recvedByte}, sctp.PayloadTypeWebRTCBinary)
			_, err = stream1.WriteDataChannel([]byte{byte(pingSeqNum % 255), recvedByte}, false)
			if err != nil {
				//log.Panic(err)
				panic(err)
			}
			fmt.Println("sent:", pingSeqNum%255, recvedByte)
			pingSeqNum++

			// test of close
			if pingSeqNum == 5 {
				stream1.Close()
				stream2.Close()
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
				a2_2.Shutdown(ctx)
				panic("Shutdown finished")
			}

			buff := make([]byte, 1024)
			_, _, err = stream2.ReadDataChannel(buff)
			if err != nil {
				//log.Panic(err)
				panic(err)
			}
			fmt.Println("received:", buff[0], buff[1])
		}
	} else if p.GossipDataMan.Self == 3 { // writer
		var pingSeqNum int
		for {
			//_, err = stream.WriteSCTP([]byte{byte(pingSeqNum % 255), recvedByte}, sctp.PayloadTypeWebRTCBinary)
			n, err := stream1.WriteDataChannel([]byte{byte(pingSeqNum % 255), recvedByte}, false)
			if err != nil || n != 2 {
				//log.Panic(err)
				panic(err)
			}
			fmt.Println("sent:", pingSeqNum%255, recvedByte)
			pingSeqNum++

			time.Sleep(3 * time.Second)
		}
	} else {
		panic("invalid destname")
	}

	//go func() {
	//	var pingSeqNum int
	//	for {
	//		_, err = stream.WriteSCTP([]byte{byte(pingSeqNum % 255), recvedByte}, sctp.PayloadTypeWebRTCBinary)
	//		if err != nil {
	//			//log.Panic(err)
	//			panic(err)
	//		}
	//
	//		fmt.Println("sent:", pingSeqNum%255, recvedByte)
	//
	//		pingSeqNum++
	//
	//		time.Sleep(3 * time.Second)
	//	}
	//}()
	//
	//for {
	//	buff := make([]byte, 1024)
	//	_, _, err = stream.ReadSCTP(buff)
	//	if err != nil {
	//		//log.Panic(err)
	//		panic(err)
	//	}
	//	fmt.Println("received:", buff[0], buff[1])
	//}
}
