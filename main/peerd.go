package main

import (
	"flag"
	"fmt"
	"github.com/ryogrid/gossip-overlay/core"
	"github.com/ryogrid/gossip-overlay/util"
	"github.com/weaveworks/mesh"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	peers := &util.Stringset{}
	var (
		httpListen = flag.String("http", ":8080", "HTTP listen address")
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

	parsed, err1 := strconv.ParseUint(*destname, 10, 64)
	if err1 != nil {
		logger.Fatalf("Could not parse Destname: %v", err)
	}

	p := core.NewPeer(name, logger, mesh.PeerName(parsed), nickname, channel, meshListen, &meshConf, peers)

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
	go func() {
		logger.Printf("HTTP server starting (%s)", *httpListen)
		http.HandleFunc("/", Handle(p))
		errs <- http.ListenAndServe(*httpListen, nil)
	}()

	logger.Print(<-errs)
}

func Handle(p *core.Peer) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			fmt.Fprintf(w, "ReadPeer => %d\n", p.ReadPeer(mesh.PeerName(3)))

		case "POST":
			for {
				// wait until routing info to Destname is available
				time.Sleep(1 * time.Millisecond)
				fetched := p.Router.Peers.Fetch(p.Destname)
				if fetched != nil {
					break
				}
			}
			fmt.Fprintf(w, "WritePeer => %d\n", p.WritePeer())
		}
	}
}
