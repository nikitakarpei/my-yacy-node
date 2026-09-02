package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	markdowncorporagrpc "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdowncorpora/grpc"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	pagereadprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pagereadprogressobservers/applog"
	pagereadprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pagereadprogressobservers/prometheus"
	scrapeoutcomesnats "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomes/nats"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scraperequests/jetstream"
	searchenginessearxng "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/searxng"
	searchprogressobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchprogressobservers/applog"
	searchprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchprogressobservers/prometheus"
	toolcallreceiversmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/toolcallreceivers/mcp"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const (
	readHeaderLimit = 10 * time.Second
	shutdownLimit   = 15 * time.Second
)

func RunService(ctx context.Context, cfg ServiceConfig) error {
	corpus, err := markdowncorporagrpc.OpenMarkdownCorpus(
		cfg.CorpusMarkdownAddr,
		cfg.CorpusMarkdownRecallDeadline,
	)
	if err != nil {
		return err
	}
	defer func() { _ = corpus.Close() }()
	scrapeRequests, err := scraperequestsjetstream.OpenScrapeRequests(cfg.ScrapeRequestNATSURL)
	if err != nil {
		return err
	}
	defer scrapeRequests.Close()
	scrapeOutcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(cfg.ScrapeRequestNATSURL)
	if err != nil {
		return err
	}
	defer scrapeOutcomes.Close()

	registry := prometheus.NewRegistry()
	search := websearch.NewWebSearch(websearch.Config{
		Engine: searchenginessearxng.NewSearXNG(cfg.SearXNGURL, cfg.SearXNGSearchDeadline),
		Progress: websearch.SearchProgressObservers{
			searchprogressobserversapplog.SearchProgressLog{},
			searchprogressobserversprometheus.New(registry),
		},
		SearchResultLimit: cfg.SearchResultLimit,
	})
	pageReader := pageread.NewPageReader(pageread.Config{
		Corpus:         corpus,
		ScrapeRequests: scrapeRequests,
		ScrapeOutcomes: scrapeOutcomes,
		Progress: pageread.PageReadProgressObservers{
			pagereadprogressobserversapplog.PageReadProgressLog{},
			pagereadprogressobserversprometheus.New(registry),
		},
		CharacterLimit:  cfg.PageFetchCharacterLimit,
		ScrapeTolerance: cfg.PageScrapeTolerance,
		FetchWait:       cfg.PageFetchWait,
	})

	toolCallServer := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: toolcallreceiversmcp.NewToolCallMux(toolcallreceiversmcp.Config{
			Search:              search,
			PageReader:          pageReader,
			ServiceVersion:      version,
			ToolCallConcurrency: cfg.ToolCallConcurrency,
		}),
		ReadHeaderTimeout: readHeaderLimit,
	}
	opsServer := &http.Server{
		Addr:              cfg.OpsAddr,
		Handler:           opsmetrics.NewMux(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, "webresearchmcp started",
		slog.String("listen", cfg.ListenAddr),
		slog.String("searxng", cfg.SearXNGURL),
		slog.String("corpus", cfg.CorpusMarkdownAddr),
		slog.Int("toolCallConcurrency", cfg.ToolCallConcurrency),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "tools", Server: toolCallServer},
		{Name: "ops", Server: opsServer},
	})
	slog.InfoContext(ctx, "webresearchmcp stopped")
	return err
}
