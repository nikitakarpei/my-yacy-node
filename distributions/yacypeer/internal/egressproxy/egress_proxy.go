// Package egressproxy runs the smokescreen proxy built into the peer. It listens on the
// loopback interface only and refuses requests to loopback and private addresses. It carries a
// request for an address of an overlay network that the peer joins itself, through that
// overlay.
package egressproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
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

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type Overlay struct {
	Addresses netip.Prefix
	Dialer    Dialer
}

type EgressProxy struct {
	listener net.Listener
	overlays []Overlay
}

func Listen(ctx context.Context, overlays []Overlay) (*EgressProxy, error) {
	listener, err := loopback.Listen(ctx)
	if err != nil {
		return nil, fmt.Errorf("built-in egress proxy: %w", err)
	}
	return &EgressProxy{listener: listener, overlays: overlays}, nil
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
			Handler:           overlayFrontFor(p.overlays, smokescreen.BuildProxy(config)),
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
	return config
}
