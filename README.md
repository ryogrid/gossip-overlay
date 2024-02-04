# gossip-overlay
## Feature
- Bi-directional reliable stream I/F which hides peers on gossip protcol layer with SCTP
- Any centralized server is not needed
- TCP socket programming style interface (system architecture like a server and clients can be implemented) 
- NAT transparent communication

## Build (sample program)
$ go build -o streamer main/streamer.go

## Usage (sample program)
Message Ping Pong between 2 peers through one intermediate peer  
(thid peer-c establishes two connectionns to peer-b on overlay NW and send ping to each connection)

Start several peers on the same host.  
Tell the second and subsequent peers to connect to the first one.  
Procedure below needs shell x 3 :)

```
$ ./streamer -hwaddr 00:00:00:00:00:01 -nickname a -mesh :6001
$ ./streamer -side recv -hwaddr 00:00:00:00:00:02 -nickname b -mesh :6002 -peer 127.0.0.1:6001
$ ./streamer -side send -hwaddr 00:00:00:00:00:03 -nickname c -mesh :6003 -peer 127.0.0.1:6001 -destid 2
```

Network Topology:  
- Peer-b <-> Peer-a <-> Peer-c 

## Use Cases
- Port forwarding tool between private networks
  - [gossip-port-forward](https://github.com/ryogrid/gossip-port-forward)
- DHT based very simple distributed KVS whose nodes run on overlay
  - [gord-overlay](https://github.com/ryogrid/gord-overlay)

## TODO
- Replace [pion/sctp](https://github.com/pion/sctp) (SCTP) with [hashicorp/yamux](https://github.com/hashicorp/yamux) (QUIC like protcol) and redesign overlay connection establishment sequence...
- [gist (in Japanese)](https://gist.github.com/ryogrid/e78088bc531bc62c10eba1c0d0e0b7fc)
