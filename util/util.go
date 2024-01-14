package util

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/ryogrid/gossip-overlay/overlay_setting"
	"net"
	"os"
	"sort"
	"strings"
)

type Stringset map[string]struct{}

func (ss Stringset) Set(value string) error {
	ss[value] = struct{}{}
	return nil
}

func (ss Stringset) String() string {
	return strings.Join(ss.Slice(), ",")
}

func (ss Stringset) Slice() []string {
	slice := make([]string, 0, len(ss))
	for k := range ss {
		slice = append(slice, k)
	}
	sort.Strings(slice)
	return slice
}

func MustHardwareAddr() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range ifaces {
		if s := iface.HardwareAddr.String(); s != "" {
			return s
		}
	}
	panic("no valid network interfaces")
}

func MustHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

func OverlayDebugPrintln(a ...interface{}) {
	if overlay_setting.OVERLAY_DEBUG {
		fmt.Println(a)
	}
}

func NewHashIDUint64(key string) uint64 {
	hf := sha256.New()
	hf.Write([]byte(key))
	//var ret uint64
	//binary.Read(bytes.NewBuffer(hf.Sum(nil)[0:7]), binary.LittleEndian, &ret)
	//return ret
	return binary.LittleEndian.Uint64(hf.Sum(nil))
}

func NewHashIDUint16(key string) uint16 {
	hf := sha256.New()
	hf.Write([]byte(key))
	//var ret uint64
	//binary.Read(bytes.NewBuffer(hf.Sum(nil)[0:7]), binary.LittleEndian, &ret)
	//return ret
	return binary.LittleEndian.Uint16(hf.Sum(nil))
}
