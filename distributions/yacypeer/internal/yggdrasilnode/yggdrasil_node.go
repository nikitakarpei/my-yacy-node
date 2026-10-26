// Package yggdrasilnode runs the Yggdrasil node of the peer. The node joins the Yggdrasil
// network through the peers that the operator names, takes peerings on the listen URLs that
// the operator names, and carries the IPv6 packets of the peer from and to its Yggdrasil address.
package yggdrasilnode

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"net/netip"
	"net/url"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"
	"github.com/yggdrasil-network/yggdrasil-go/src/ipv6rwc"
)

const (
	Name                = "yggdrasil"
	networkPrefixLength = 7
	anySourceInterface  = ""
)

var NetworkAddresses = netip.PrefixFrom(
	netip.AddrFrom16([16]byte{address.GetPrefix()[0]}),
	networkPrefixLength,
)

type Peering struct {
	PeerURLs   []*url.URL
	ListenURLs []*url.URL
}

type Node struct {
	core    *core.Core
	packets *ipv6rwc.ReadWriteCloser
}

func Start(ctx context.Context, key ed25519.PrivateKey, peering Peering) (*Node, error) {
	nodeConfig := &config.NodeConfig{PrivateKey: config.KeyBytes(key)}
	if err := nodeConfig.GenerateSelfSignedCertificate(); err != nil {
		return nil, fmt.Errorf("yggdrasil certificate: %w", err)
	}
	yggdrasilCore, err := core.New(
		nodeConfig.Certificate,
		nodeLogFor(ctx),
		peerOptionsFor(peering)...)
	if err != nil {
		return nil, fmt.Errorf("yggdrasil: %w", err)
	}
	for _, listenURL := range peering.ListenURLs {
		if _, err := yggdrasilCore.Listen(listenURL, anySourceInterface); err != nil {
			yggdrasilCore.Stop()
			return nil, fmt.Errorf(
				"yggdrasil: listen for peers on %s: %w",
				listenURL.Redacted(),
				err,
			)
		}
	}
	return &Node{core: yggdrasilCore, packets: ipv6rwc.NewReadWriteCloser(yggdrasilCore)}, nil
}

func peerOptionsFor(peering Peering) []core.SetupOption {
	options := make([]core.SetupOption, 0, len(peering.PeerURLs))
	for _, peerURL := range peering.PeerURLs {
		options = append(options, core.Peer{URI: peerURL.String()})
	}
	return options
}

func (n *Node) Address() netip.Addr {
	nodeAddress, _ := netip.AddrFromSlice(n.core.Address())
	return nodeAddress
}

func (n *Node) MTU() uint32 {
	return uint32(n.packets.MTU()) //nolint:gosec // G115: ipv6rwc caps the MTU at 65535.
}

func (n *Node) Read(packet []byte) (int, error) {
	size, err := n.packets.Read(packet)
	if err != nil {
		return size, fmt.Errorf("read a Yggdrasil packet: %w", err)
	}
	return size, nil
}

func (n *Node) Write(packet []byte) (int, error) {
	size, err := n.packets.Write(packet)
	if err != nil {
		return size, fmt.Errorf("write a Yggdrasil packet: %w", err)
	}
	return size, nil
}

func (n *Node) Stop() {
	n.core.Stop()
}
