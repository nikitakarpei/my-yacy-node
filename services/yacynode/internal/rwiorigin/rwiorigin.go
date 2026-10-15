// Package rwiorigin names where inbound RWI postings come from: the peer hash
// the sending peer claims and the network it sends from. Two transfers share an
// origin only when both match.
package rwiorigin

import (
	"net/netip"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	ipv4NetworkBits = 24
	ipv6NetworkBits = 56
)

type Origin struct {
	Peer    yacymodel.Hash
	Network string
}

func Of(peer yacymodel.Hash, address string) Origin {
	return Origin{Peer: peer, Network: networkOf(address)}
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
	stored.Bytes(origin.Peer.AppendBytes(nil))
	stored.Text(origin.Network)
}

func From(stored *storedfields.Reader) Origin {
	peer := stored.Bytes("origin peer")
	origin := Origin{Network: stored.Text("origin network")}
	if len(peer) == 0 {
		return origin
	}
	parsed, err := yacymodel.ParseHashBytes(peer)
	if err != nil {
		stored.Reject("origin peer", err)

		return Origin{}
	}
	origin.Peer = parsed

	return origin
}
