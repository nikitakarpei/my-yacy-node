package egressproxy

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/stripe/smokescreen/pkg/smokescreen"
)

var defaultPorts = map[string]uint16{"http": 80, "https": 443}

type upstreamProxy struct {
	url       *url.URL
	addresses []netip.AddrPort
}

func resolveUpstreamProxy(ctx context.Context, upstreamProxyURL *url.URL) upstreamProxy {
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", upstreamProxyURL.Hostname())
	if err != nil {
		slog.WarnContext(ctx, "upstream proxy not resolved",
			slog.String("upstreamProxy", upstreamProxyURL.Redacted()),
			slog.Any("reason", err),
		)
		return upstreamProxy{url: upstreamProxyURL}
	}
	port := portOf(upstreamProxyURL)
	addresses := make([]netip.AddrPort, 0, len(ips))
	for _, ip := range ips {
		addresses = append(addresses, netip.AddrPortFrom(ip.Unmap(), port))
	}
	return upstreamProxy{url: upstreamProxyURL, addresses: addresses}
}

func portOf(upstreamProxyURL *url.URL) uint16 {
	if port, err := strconv.ParseUint(upstreamProxyURL.Port(), 10, 16); err == nil {
		return uint16(port)
	}
	return defaultPorts[upstreamProxyURL.Scheme]
}

func (u upstreamProxy) addressRules() []smokescreen.RuleRange {
	rules := make([]smokescreen.RuleRange, 0, len(u.addresses))
	for _, address := range u.addresses {
		bits := address.Addr().BitLen()
		rules = append(rules, smokescreen.RuleRange{
			Net:  net.IPNet{IP: address.Addr().AsSlice(), Mask: net.CIDRMask(bits, bits)},
			Port: int(address.Port()),
		})
	}
	return rules
}

func (u upstreamProxy) listensAt(address string) bool {
	selected, _ := netip.ParseAddrPort(address)
	return slices.Contains(
		u.addresses,
		netip.AddrPortFrom(selected.Addr().Unmap(), selected.Port()),
	)
}

func (u upstreamProxy) dial(
	ctx context.Context,
	network string,
	timeout time.Duration,
) (net.Conn, error) {
	failures := make([]error, 0, len(u.addresses))
	for _, address := range u.addresses {
		connection, err := dial(ctx, network, address.String(), timeout)
		if err == nil {
			return connection, nil
		}
		failures = append(failures, err)
	}
	return nil, errors.Join(failures...)
}
