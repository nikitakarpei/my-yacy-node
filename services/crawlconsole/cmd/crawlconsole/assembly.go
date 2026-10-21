package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	consolepageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolepageobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolepages"
	crawlershttp "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/crawlers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "crawlconsole started"
	msgServiceStopped = "crawlconsole stopped"
)

type Listeners struct {
	Console net.Listener
	Ops     net.Listener
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	crawler := crawlershttp.New(cfg.CrawlerAPIURL, cfg.CrawlerCallDeadline())
	consoleServer := &http.Server{
		Handler: httpobservation.NewHandler(
			consolepages.New(
				crawler,
				consolemessages.English(),
				wallclock.Clock{},
				consolepageobserversapplog.ConsoleLog{},
			),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "crawlconsole"),
		),
		ReadHeaderTimeout: readHeaderLimit,
	}
	opsServer := &http.Server{
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Console.Addr().String()),
		slog.String("crawlerApiUrl", cfg.CrawlerAPIURL.String()),
	)

	err := servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "console", Listener: listeners.Console, Server: consoleServer},
		{Name: "ops", Listener: listeners.Ops, Server: opsServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
