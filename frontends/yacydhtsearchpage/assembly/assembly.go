package yacydhtsearchpageassembly

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchengines/yacysearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchmessages"
	searchpageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchpageobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchpages"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "yacydhtsearchpage started"
	msgServiceStopped = "yacydhtsearchpage stopped"
)

type Listeners struct {
	Page net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	page, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Page: page}, nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "yacydhtsearchpage_info",
		Help: "YaCy DHT search page application identity.",
	}, func() float64 { return 1 }))
	pageServer := &http.Server{
		Handler: httpobservation.NewHandler(
			searchpages.New(
				yacysearch.New(cfg.YaCyDHTSearchURL, cfg.SearchCallDeadline()),
				searchmessages.English(),
				searchpageobserversapplog.SearchPageLog{},
			),
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "yacydhtsearchpage"),
		),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Page.Addr().String()),
		slog.String("yacydhtsearchUrl", cfg.YaCyDHTSearchURL.String()),
	)

	err := servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "page", Listener: listeners.Page, Server: pageServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}
