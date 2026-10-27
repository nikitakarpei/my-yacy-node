package egressproxy

import (
	"context"
	"errors"
	"fmt"
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

func upstreamProxyFor(ctx context.Context, upstreamProxyURL *url.URL) upstreamProxy {
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

func (u upstreamProxy) ruleRanges() []smokescreen.RuleRange {
	ranges := make([]smokescreen.RuleRange, 0, len(u.addresses))
	for _, address := range u.addresses {
		bits := address.Addr().BitLen()
		ranges = append(ranges, smokescreen.RuleRange{
			Net:  net.IPNet{IP: address.Addr().AsSlice(), Mask: net.CIDRMask(bits, bits)},
			Port: int(address.Port()),
		})
	}
	return ranges
}

func (u upstreamProxy) dial(
	ctx context.Context,
	network string,
	address string,
	timeout time.Duration,
) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}
	selected, _ := netip.ParseAddrPort(address)
	if !slices.Contains(
		u.addresses,
		netip.AddrPortFrom(selected.Addr().Unmap(), selected.Port()),
	) {
		connection, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", address, err)
		}
		return connection, nil
	}
	failures := make([]error, 0, len(u.addresses))
	for _, upstreamAddress := range u.addresses {
		connection, err := dialer.DialContext(ctx, network, upstreamAddress.String())
		if err == nil {
			return connection, nil
		}
		failures = append(failures, err)
	}
	return nil, errors.Join(failures...)
}
