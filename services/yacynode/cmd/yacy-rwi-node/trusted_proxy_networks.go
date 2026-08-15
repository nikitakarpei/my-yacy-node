package main

import (
	"fmt"
	"net"
	"strings"
)

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
