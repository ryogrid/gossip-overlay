# gossip-overlay
## Feature
- Bi-directional reliable stream I/F which hides peers on gossip protcol layer with SCTP
- Any centralized server is not needed
- TCP socket programming style interface (system architecture like a server and clients can be implemented) 
- Machines on network having NAT can join

## Build (sample program)
$ go build main/streamer.go

## Usage
Message Ping Pong between 2 peers through one intermediate peer  

Start several peers on the same host.  
Tell the second and subsequent peers to connect to the first one.  
Procedure below needs shell x 3 :)

```
$ ./streamer -hwaddr 00:00:00:00:00:01 -nickname a -mesh :6001
$ ./streamer -side recv -hwaddr 00:00:00:00:00:02 -nickname b -mesh :6002 -peer 127.0.0.1:6001
$ ./streamer -side send -hwaddr 00:00:00:00:00:03 -nickname c -mesh :6003 -peer 127.0.0.1:6001 -destname 2
```

Network Topology:  
- Peer-b <-> Peer-a <-> Peer-c 

https://github.com/ryogrid/gossip-overlay/assets/24614/176837aa-bcdf-414a-a864-5feb63eab3a1

## Use Cases
- Port forwarding tool
  - [gossip-port-forward](https://github.com/ryogrid/gossip-port-forward)
- DHT based distributed KVS
  - IN PROGRESS

## TODO
- [gist (in Japanese)](https://gist.github.com/ryogrid/e78088bc531bc62c10eba1c0d0e0b7fc)
