package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/ryogrid/gossip-overlay/overlay"
	"github.com/ryogrid/gossip-overlay/overlay_setting"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
)

/*
.\streamer.exe -side recv -hwaddr 00:00:00:00:00:02 -nickname b -mesh :6002 -debug false | Tee-Object -FilePath ".\recv-1.txt"
.\streamer.exe -side send -hwaddr 00:00:00:00:00:03 -nickname c -mesh :6003 -destid 2 -peer 127.0.0.1:6002 -debug false | Tee-Object -FilePath ".\send-1.txt"
*/
func main() {
	peers := &util.Stringset{}
	var (
		side       = flag.String("side", "relay", "specify peer type (default: relay))")
		meshListen = flag.String("mesh", net.JoinHostPort("0.0.0.0", strconv.Itoa(mesh.Port)), "mesh listen address")
		hwaddr     = flag.String("hwaddr", util.MustHardwareAddr(), "MAC address style number description for generating self peer ID (default: MAC address of primary NW interface)")
		nickname   = flag.String("nickname", util.MustHostname(), "peer nickname (default: hostname of launched machine)")
		destId     = flag.String("destid", "", "destination peer peerId (optional)")
		debug      = flag.String("debug", "false", "print debug info, true of false (default: false)")
	)
	flag.Var(peers, "peer", "initial peer (may be repeated)")
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

	runtime.GOMAXPROCS(10)
	peerId, err := mesh.PeerNameFromString(*hwaddr)
	if err != nil {
		logger.Fatalf("%s: %v", *hwaddr, err)
	}

	p, err := overlay.NewOverlayPeer(uint64(peerId), &host, port, peers, false)
	if err != nil {
		panic(err)
	}

	var destIdNum uint64 = math.MaxUint64
	if *destId != "" {
		destIdNum, err = strconv.ParseUint(*destId, 10, 64)
		if err != nil {
			logger.Fatalf("Could not parse Destname: %v", err)
		}
	}

	errs := make(chan error)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	if *side == "send" {
		go clientRoutine(p, mesh.PeerName(destIdNum))
		go clientRoutine(p, mesh.PeerName(destIdNum))
	} else if *side == "recv" {
		go serverRoutine(p)
	} else {
		// do nothing (relay)
	}

	logger.Print(<-errs)
}

func serverRoutine(p *overlay.OverlayPeer) {
	util.OverlayDebugPrintln("start serverRoutine")

	for {
		channel, remotePeerId, remoteHost, _, err := p.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("accepted:", uint64(remotePeerId), remoteHost)

		go func(channel_ net.Conn) {
			pongSeqNum := 0
			for {
				util.OverlayDebugPrintln("call ReadDataChannel!")
				buff := make([]byte, 1024)
				n, err := channel_.Read(buff)
				if err != nil || n != 1 {
					util.OverlayDebugPrintln("panic occured at ReadDataChannel!", err, n)
					panic(err)
				}
				fmt.Println("received:", buff[0])

				util.OverlayDebugPrintln("call WriteDataChannel!")
				n, err = channel_.Write([]byte{byte(pongSeqNum % 255), buff[0]})
				if err != nil || n != 2 {
					panic(err)
				}
				fmt.Println("sent:", pongSeqNum%255, buff[0])
				pongSeqNum++
			}
		}(channel)
	}
}

func clientRoutine(p *overlay.OverlayPeer, destPeerId mesh.PeerName) {
	util.OverlayDebugPrintln("start clientRoutine")

	// second argument is for specific application...
	// you can set empty string
	channel := p.OpenStreamToTargetPeer(destPeerId, "")

	pingSeqNum := 0
	for {
		util.OverlayDebugPrintln("call WriteDataChannel!")
		n, err := channel.Write([]byte{byte(pingSeqNum % 255)})
		if err != nil || n != 1 {
			util.OverlayDebugPrintln("panic occured at WriteDataChannel!", err, n)
			panic(err)
		}
		fmt.Println("sent:", pingSeqNum%255)
		pingSeqNum++

		util.OverlayDebugPrintln("call ReadDataChannel!")
		buff := make([]byte, 1024)
		n, err = channel.Read(buff)
		if err != nil || n != 2 {
			util.OverlayDebugPrintln("panic occured at ReadDataChannel!", err, n)
			panic(err)
		}
		fmt.Println("received:", buff[0], buff[1])

		time.Sleep(3 * time.Second)
	}
}
