// Package rwiorigin names where inbound RWI postings come from: the network
// the sending peer sends from. Two transfers share an origin when they come
// from one network.
package rwiorigin

import (
	"net/netip"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
)

const (
	ipv4NetworkBits = 24
	ipv6NetworkBits = 56
)

type Origin struct {
	Network string
}

func Of(address string) Origin {
	return Origin{Network: networkOf(address)}
}

func networkOf(address string) string {
	ip, err := netip.ParseAddr(address)
	if err != nil {
		return address
	}
	ip = ip.Unmap().WithZone("")
	networkBits := ipv6NetworkBits
	if ip.Is4() {
		networkBits = ipv4NetworkBits
	}

	return netip.PrefixFrom(ip, networkBits).Masked().String()
}

func Write(stored *storedfields.Writer, origin Origin) {
	stored.Text(origin.Network)
}

func From(stored *storedfields.Reader) Origin {
	return Origin{Network: stored.Text("origin network")}
}
