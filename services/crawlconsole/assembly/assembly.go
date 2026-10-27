package crawlconsoleassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolemessages"
	consolepageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolepageobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/consolepages"
	crawlershttp "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/internal/crawlers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
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
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	console, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Console: console}, nil
}

func (l Listeners) Close() error {
	if err := l.Console.Close(); err != nil {
		return fmt.Errorf("close the console listener: %w", err)
	}
	return nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "crawlconsole_info",
		Help: "Crawl console application identity.",
	}, func() float64 { return 1 }))
	crawler := crawlershttp.New(cfg.CrawlerAPIURL, cfg.CrawlerCallDeadline())
	consoleServer := &http.Server{
		Handler: httpobservation.NewHandler(
			consolepages.New(
				crawler,
				consolemessages.English(),
				cfg.TabBar,
				wallclock.Clock{},
				consolepageobserversapplog.ConsoleLog{},
			),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "crawlconsole"),
		),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Console.Addr().String()),
		slog.String("crawlerApiUrl", cfg.CrawlerAPIURL.String()),
	)

	err := servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "console", Listener: listeners.Console, Server: consoleServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
