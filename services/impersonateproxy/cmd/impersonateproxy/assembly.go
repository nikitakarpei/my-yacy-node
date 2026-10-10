package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/chromenavigation"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/pagefetchers/tlsclient"
	tlsclientobserversapplog "github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/pagefetchersobservers/tlsclient/applog"
	tlsclientobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/pagefetchersobservers/tlsclient/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintake"
	proxyintakeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintakeobservers/applog"
	proxyintakeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintakeobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "impersonateproxy started"
	msgServiceStopped = "impersonateproxy stopped"
)

func RunService(ctx context.Context, cfg ServiceConfig, registry *prometheus.Registry) error {
	trustedRoots, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("load system roots: %w", err)
	}
	pageFetcher, err := tlsclient.New(
		cfg.EgressProxyURL,
		trustedRoots,
		cfg.FetchTimeout,
		tlsclient.Observers{
			tlsclientobserversapplog.FetchLog{},
			tlsclientobserversprometheus.New(registry),
		},
	)
	if err != nil {
		return err
	}
	proxyServer := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: proxyintake.New(
			chromenavigation.New(cfg.DefaultReferer),
			pageFetcher,
			proxyintake.Observers{
				proxyintakeobserversapplog.IntakeLog{},
				proxyintakeobserversprometheus.New(registry),
			},
		),
		ReadHeaderTimeout: readHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", cfg.ListenAddr),
		slog.Int("chromeVersion", chromenavigation.MajorVersion),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "proxy", Server: proxyServer},
		{Name: "ops", Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
