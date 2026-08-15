package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/cdprender"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/originpreflight"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/proxyintake"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendermetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	opsReadHeaderLimit = 10 * time.Second
	shutdownLimit      = 15 * time.Second
	msgServiceStarted  = "renderproxy started"
	msgServiceStopped  = "renderproxy stopped"
)

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	registry *prometheus.Registry,
) error {
	metrics := rendermetrics.New(registry)
	browser := cdprender.New(ctx, cfg.CDPURL, cfg.MaxResponseBytes)
	defer browser.Close()

	gated := rendergate.New(
		originpreflight.New(browser, cfg.EgressProxyURL, cfg.MaxResponseBytes),
		cfg.RenderConcurrency,
		cfg.RequestDeadline,
		metrics,
	)

	proxyServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           proxyintake.New(gated),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: opsReadHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", cfg.ListenAddr),
		slog.Int("renderConcurrency", cfg.RenderConcurrency),
	)

	err := servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "proxy", Server: proxyServer},
		{Name: "ops", Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
