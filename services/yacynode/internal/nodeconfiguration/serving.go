package nodeconfiguration

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPeerAddr       = "YACY_PEER_ADDR"
	EnvSecurePeerAddr = "YACY_SECURE_PEER_ADDR"
	EnvOpsAddr        = "YACY_OPS_ADDR"
	EnvTrustedProxies = "YACY_TRUSTED_PROXIES"

	DefaultPeerAddr       = ":8090"
	DefaultSecurePeerAddr = ":8443"
	DefaultOpsAddr        = ":9090"
)

type ServingConfig struct {
	PeerAddr             string
	PeerPort             int
	SecurePeerAddr       string
	SecurePeerPort       int
	OpsAddr              string
	TrustedProxyNetworks []*net.IPNet
}

func loadServingConfig(getenv func(string) string) (ServingConfig, error) {
	trustedProxyNetworks, err := trustedProxyNetworksFrom(getenv(EnvTrustedProxies))
	if err != nil {
		return ServingConfig{}, fmt.Errorf("%s: %w", EnvTrustedProxies, err)
	}

	peerAddr := envconfig.String(getenv, EnvPeerAddr, DefaultPeerAddr)
	peerPort, err := listenPortOf(EnvPeerAddr, peerAddr)
	if err != nil {
		return ServingConfig{}, err
	}

	securePeerAddr := envconfig.String(getenv, EnvSecurePeerAddr, DefaultSecurePeerAddr)
	securePeerPort, err := listenPortOf(EnvSecurePeerAddr, securePeerAddr)
	if err != nil {
		return ServingConfig{}, err
	}

	return ServingConfig{
		PeerAddr:             peerAddr,
		PeerPort:             peerPort,
		SecurePeerAddr:       securePeerAddr,
		SecurePeerPort:       securePeerPort,
		OpsAddr:              envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		TrustedProxyNetworks: trustedProxyNetworks,
	}, nil
}

func listenPortOf(listenAddrKey, listenAddr string) (int, error) {
	_, portPart, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", listenAddrKey, err)
	}

	port, err := strconv.Atoi(portPart)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", listenAddrKey, err)
	}
	if port <= 0 {
		return 0, fmt.Errorf("%s: must carry a positive port", listenAddrKey)
	}

	return port, nil
}

func trustedProxyNetworksFrom(raw string) ([]*net.IPNet, error) {
	var networks []*net.IPNet
	for item := range strings.SplitSeq(raw, ",") {
		entry := strings.TrimSpace(item)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", entry, err)
			}
			networks = append(networks, network)

			continue
		}

		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP %q", entry)
		}
		networks = append(networks, hostNetwork(ip))
	}

	return networks, nil
}

func hostNetwork(ip net.IP) *net.IPNet {
	if v4 := ip.To4(); v4 != nil {
		return &net.IPNet{IP: v4, Mask: net.CIDRMask(32, 32)}
	}

	return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
}
