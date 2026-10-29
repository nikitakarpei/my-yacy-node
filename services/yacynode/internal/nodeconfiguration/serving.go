package nodeconfiguration

import (
	"fmt"
	"net"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	EnvPeerAddr          = "YACY_PEER_ADDR"
	EnvSecurePeerEnabled = "YACY_SECURE_PEER_ENABLED"
	EnvSecurePeerAddr    = "YACY_SECURE_PEER_ADDR"
	EnvOpsAddr           = "YACY_OPS_ADDR"
	EnvTrustedProxies    = "YACY_TRUSTED_PROXIES"

	DefaultPeerAddr          = ":8090"
	DefaultSecurePeerEnabled = true
	DefaultSecurePeerAddr    = ":8443"
	DefaultOpsAddr           = ":9090"
)

type ServingConfig struct {
	PeerAddr             string
	SecurePeerAddr       yacymodel.Optional[string]
	OpsAddr              string
	TrustedProxyNetworks []*net.IPNet
}

func loadServingConfig(getenv func(string) string) (ServingConfig, error) {
	trustedProxyNetworks, err := trustedProxyNetworksFrom(getenv(EnvTrustedProxies))
	if err != nil {
		return ServingConfig{}, fmt.Errorf("%s: %w", EnvTrustedProxies, err)
	}

	securePeerAddr, err := securePeerAddrOf(getenv)
	if err != nil {
		return ServingConfig{}, err
	}

	return ServingConfig{
		PeerAddr:             envconfig.String(getenv, EnvPeerAddr, DefaultPeerAddr),
		SecurePeerAddr:       securePeerAddr,
		OpsAddr:              envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		TrustedProxyNetworks: trustedProxyNetworks,
	}, nil
}

func securePeerAddrOf(getenv func(string) string) (yacymodel.Optional[string], error) {
	enabled, err := envconfig.Bool(getenv, EnvSecurePeerEnabled, DefaultSecurePeerEnabled)
	if err != nil {
		return yacymodel.None[string](), err
	}
	if !enabled {
		return yacymodel.None[string](), nil
	}

	return yacymodel.Some(envconfig.String(getenv, EnvSecurePeerAddr, DefaultSecurePeerAddr)), nil
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
