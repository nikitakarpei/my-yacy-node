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
	rendercapacityobserversapplog "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendercapacityobservers/applog"
	rendercapacityobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendercapacityobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
	renderobserversapplog "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderobservers/applog"
	renderobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
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
	renderObservers := rendergate.RenderObservers{
		renderobserversapplog.RenderLog{},
		renderobserversprometheus.New(registry),
	}
	renderCapacityObservers := rendergate.RenderCapacityObservers{
		rendercapacityobserversapplog.RenderCapacityLog{},
		rendercapacityobserversprometheus.New(registry),
	}
	browser := cdprender.New(ctx, cfg.CDPURL, cfg.EgressProxyURL, cfg.MaxResponseBytes)
	defer browser.Close()

	deadlineRenderer := rendergate.NewDeadlineRenderer(
		originpreflight.New(browser, cfg.EgressProxyURL, cfg.MaxResponseBytes),
		cfg.RequestDeadline,
		renderObservers,
	)
	capacityLimitedRenderer := rendergate.NewCapacityLimitedRenderer(
		deadlineRenderer,
		cfg.RenderConcurrency,
		renderCapacityObservers,
	)
	proxyMux := http.NewServeMux()
	proxyMux.Handle("/", proxyintake.New(capacityLimitedRenderer))

	proxyServer := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: httpobservation.NewHandler(
			proxyMux,
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "renderproxy"),
		),
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
