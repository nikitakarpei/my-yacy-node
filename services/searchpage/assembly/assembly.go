package searchpageassembly

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	resultlinksdirect "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultlinks/direct"
	resultlinksvisitcrawl "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultlinks/visitcrawl"
	resultsourceobserversapplog "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultsourceobservers/applog"
	resultsourcescorpustext "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultsources/corpustext"
	resultsourcesyacysearch "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultsources/yacysearch"
	searchpageobserversapplog "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchpageobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchpages"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/visitrelay"
	visitrelayobserversapplog "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/visitrelayobservers/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpaccesslog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
)

const (
	readHeaderLimit   = 10 * time.Second
	shutdownLimit     = 15 * time.Second
	msgServiceStarted = "searchpage started"
	msgServiceStopped = "searchpage stopped"
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

func (l Listeners) Close() error {
	if err := l.Page.Close(); err != nil {
		return fmt.Errorf("close the page listener: %w", err)
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
		Name: "searchpage_info",
		Help: "Search page application identity.",
	}, func() float64 { return 1 }))
	mux := http.NewServeMux()
	mountVisitRelay(mux, cfg.VisitCrawl)
	mux.Handle("/", searchpages.New(
		resultSourcesFor(cfg),
		resultLinksFor(cfg.VisitCrawl),
		cfg.TabBar,
		searchpageobserversapplog.SearchPageLog{},
	))
	pageServer := &http.Server{
		Handler: httpobservation.NewHandler(
			mux,
			httpaccesslog.New(),
			httpmetrics.NewEndpointMetrics(registry, "searchpage"),
		),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, msgServiceStarted,
		slog.String("listenAddr", listeners.Page.Addr().String()),
		slog.String("corpusTextUrl", textOf(cfg.CorpusTextURL)),
		slog.String("yacySearchUrl", textOf(cfg.YaCySearchURL)),
		slog.Bool("linksThroughVisitCrawl", cfg.VisitCrawl != nil),
	)

	err := servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "page", Listener: listeners.Page, Server: pageServer},
	})
	slog.InfoContext(ctx, msgServiceStopped)
	return err
}

func mountVisitRelay(mux *http.ServeMux, visitCrawl *VisitCrawlConfig) {
	if visitCrawl == nil {
		return
	}
	mux.Handle("GET "+resultlinksvisitcrawl.Path, visitrelay.New(
		visitCrawl.URL,
		visitrelayobserversapplog.VisitRelayLog{},
	))
}

func resultSourcesFor(cfg ServiceConfig) map[searchqueries.Source]searchpages.ResultSource {
	sources := map[searchqueries.Source]searchpages.ResultSource{}
	if cfg.CorpusTextURL != nil {
		sources[searchqueries.CrawledPages] = resultsourcescorpustext.New(
			cfg.CorpusTextURL,
			cfg.CorpusTextCallDeadline(),
			resultsourceobserversapplog.CorpusTextLog{},
		)
	}
	if cfg.YaCySearchURL != nil {
		sources[searchqueries.YaCyNetwork] = resultsourcesyacysearch.New(
			cfg.YaCySearchURL,
			cfg.YaCySearchCallDeadline,
			resultsourceobserversapplog.YaCySearchLog{},
		)
	}
	return sources
}

func resultLinksFor(visitCrawl *VisitCrawlConfig) searchpages.ResultLinks {
	if visitCrawl == nil {
		return resultlinksdirect.New()
	}
	return resultlinksvisitcrawl.New(
		visitCrawl.LinkSecret,
		visitCrawl.LinkLifetime,
		wallclock.Clock{},
	)
}

func textOf(address *url.URL) string {
	if address == nil {
		return ""
	}
	return address.String()
}
