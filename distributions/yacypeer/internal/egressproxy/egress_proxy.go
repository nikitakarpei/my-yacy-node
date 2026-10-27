// Package egressproxy runs the smokescreen proxy built into the peer. It listens on the
// loopback interface only and refuses requests to loopback and private addresses. It sends the
// requests it allows through the upstream proxy that it is given, else to their destination. It
// reaches an address of an overlay network that the peer joins itself through that overlay.
package egressproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"github.com/stripe/smokescreen/pkg/smokescreen"
	"github.com/stripe/smokescreen/pkg/smokescreen/conntrack"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/loopback"
)

const (
	Name          = "egressproxy"
	shutdownLimit = 10 * time.Second
)

var defaultPorts = map[string]int{"http": 80, "https": 443}

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Overlay struct {
	Addresses netip.Prefix
	Dialer    Dialer
}

type EgressProxy struct {
	listener         net.Listener
	upstreamProxyURL *url.URL
	overlays         []Overlay
}

func Listen(
	ctx context.Context,
	upstreamProxyURL *url.URL,
	overlays []Overlay,
) (*EgressProxy, error) {
	listener, err := loopback.Listen(ctx)
	if err != nil {
		return nil, fmt.Errorf("built-in egress proxy: %w", err)
	}
	return &EgressProxy{
		listener:         listener,
		upstreamProxyURL: upstreamProxyURL,
		overlays:         overlays,
	}, nil
}

func (p *EgressProxy) URL() *url.URL {
	return &url.URL{Scheme: "http", Host: p.listener.Addr().String()}
}

func (p *EgressProxy) Close() error {
	if err := p.listener.Close(); err != nil {
		return fmt.Errorf("built-in egress proxy: %w", err)
	}
	return nil
}

func (p *EgressProxy) Run(ctx context.Context) error {
	config := p.configFor(ctx)
	return servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{{
		Name:     Name,
		Listener: p.listener,
		Server: &http.Server{
			Handler:           smokescreen.BuildProxy(config),
			ReadHeaderTimeout: config.ReadHeaderTimeout,
		},
	}})
}

func (p *EgressProxy) configFor(ctx context.Context) *smokescreen.Config {
	config := smokescreen.NewConfig()
	config.Log = proxyLoggerFor(ctx)
	config.ConnectTimeout = smokescreen.DefaultConnectTimeout
	config.ShuttingDown.Store(false)
	config.ConnTracker = conntrack.NewTracker(
		config.IdleTimeout,
		config.MetricsClient,
		config.Log,
		config.ShuttingDown,
		nil,
	)
	config.ProxyDialTimeout = p.dial
	if p.upstreamProxyURL != nil {
		config.UpstreamHttpProxyAddr = p.upstreamProxyURL.String()
		config.UpstreamHttpsProxyAddr = p.upstreamProxyURL.String()
		config.AllowRanges = append(config.AllowRanges, p.upstreamProxyRules()...)
	}
	return config
}

func (p *EgressProxy) dial(
	ctx context.Context,
	network string,
	address string,
	timeout time.Duration,
) (net.Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return p.dialerFor(address).DialContext(dialCtx, network, address)
}

func (p *EgressProxy) dialerFor(address string) Dialer {
	destination, _ := netip.ParseAddrPort(address)
	for _, overlay := range p.overlays {
		if overlay.Addresses.Contains(destination.Addr().Unmap()) {
			return overlay.Dialer
		}
	}
	return &net.Dialer{}
}

func (p *EgressProxy) upstreamProxyRules() []smokescreen.RuleRange {
	address, err := netip.ParseAddr(p.upstreamProxyURL.Hostname())
	if err != nil {
		return nil
	}
	address = address.Unmap()
	bits := address.BitLen()
	return []smokescreen.RuleRange{{
		Net:  net.IPNet{IP: address.AsSlice(), Mask: net.CIDRMask(bits, bits)},
		Port: p.upstreamProxyPort(),
	}}
}

func (p *EgressProxy) upstreamProxyPort() int {
	if port, err := strconv.Atoi(p.upstreamProxyURL.Port()); err == nil {
		return port
	}
	return defaultPorts[p.upstreamProxyURL.Scheme]
}
