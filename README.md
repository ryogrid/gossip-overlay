# gossip-overlay (IN PROGRESS)

## Build
$ go build main/node.go

## Usage (preview of current implemented feature)

Start several peers on the same host.  
Tell the second and subsequent peers to connect to the first one.  
Procedure below needs shell x 3 :)

```
$ ./node -hwaddr 00:00:00:00:00:01 -nickname a -mesh :6001 -http :8001 -destname 2
$ ./node -hwaddr 00:00:00:00:00:02 -nickname b -mesh :6002 -http :8002 -destname 1 -peer 127.0.0.1:6001
$ ./node -hwaddr 00:00:00:00:00:03 -nickname c -mesh :6003 -http :8003 -destname 2 -peer 127.0.0.1:6001
```

Network Topology:  
- Node-b <-> Node-a <-> Node-c 
  - In general, Node-b and Node-c may have direct TCP connection between these if there are no blocker such as NAT
  - In this example, it is not strange that the connection exists but it depends on design of mesh lib and I haven't confirmed it yet...

Write values with HTTP API (random 2 bytes are wrote from node-c to node-b).

```
$ curl -Ss -XPOST "http://127.0.0.1:8003/"
write => []
```

Read and clear datas of buffered data on Node-b which is wrote from node-c.  
(Web browser is also OK)  

```
$ curl -Ss -XGET "http://127.0.0.1:8002/"
read => [ <data on buf as byte sequence> ]
```

## TODO
- [gist (in Japanese)](https://gist.github.com/ryogrid/e78088bc531bc62c10eba1c0d0e0b7fc)
