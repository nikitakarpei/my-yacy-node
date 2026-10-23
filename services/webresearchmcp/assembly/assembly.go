package webresearchmcpassembly

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	markdowncorporahttp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdowncorpora/http"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	pagereadobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pagereadobservers/applog"
	pagereadobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pagereadobservers/prometheus"
	scrapeoutcomeobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomeobservers/applog"
	scrapeoutcomeobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomeobservers/prometheus"
	scrapeoutcomesnats "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomes/nats"
	scraperequestobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scraperequestobservers/applog"
	scraperequestobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scraperequestobservers/prometheus"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scraperequests/jetstream"
	searchenginescorpustext "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/corpustext"
	searchenginessearxng "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/searxng"
	searchenginesstack "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/stack"
	searchenginesyacy "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/yacy"
	stackobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/stackobservers/applog"
	stackobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/stackobservers/prometheus"
	toolcallreceiversmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/toolcallreceivers/mcp"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
	websearchobserversapplog "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearchobservers/applog"
	websearchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearchobservers/prometheus"
)

const (
	readHeaderLimit = 10 * time.Second
	shutdownLimit   = 15 * time.Second
)

var version = "dev"

type Listeners struct {
	Tools net.Listener
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	tools, err := servergroup.Listen(ctx, cfg.ListenAddr)
	if err != nil {
		return Listeners{}, err
	}
	return Listeners{Tools: tools}, nil
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	corpus := markdowncorporahttp.NewMarkdownCorpus(
		cfg.CorpusMarkdownAddr,
		cfg.CorpusMarkdownRecallDeadline,
	)

	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "webresearchmcp_info",
		Help: "Web research MCP application identity.",
	}, func() float64 { return 1 }))
	scrapeRequests, err := scraperequestsjetstream.OpenScrapeRequests(
		cfg.ScrapeRequestNATSURL,
		scraperequestsjetstream.ScrapeRequestPublicationObservers{
			scraperequestobserversapplog.ScrapeRequestLog{},
			scraperequestobserversprometheus.New(registry),
		},
	)
	if err != nil {
		return err
	}
	defer scrapeRequests.Close()
	scrapeOutcomes, err := scrapeoutcomesnats.OpenScrapeOutcomes(
		cfg.ScrapeRequestNATSURL,
		scrapeoutcomesnats.ScrapeOutcomeObservers{
			scrapeoutcomeobserversapplog.ScrapeOutcomeLog{},
			scrapeoutcomeobserversprometheus.New(registry),
		},
	)
	if err != nil {
		return err
	}
	defer scrapeOutcomes.Close()
	search := websearch.NewWebSearch(websearch.Config{
		Engine: webSearchEngineFor(cfg, registry),
		Observer: websearch.WebSearchObservers{
			websearchobserversapplog.WebSearchLog{},
			websearchobserversprometheus.New(registry),
		},
		SearchResultLimit: cfg.SearchResultLimit,
	})
	pageReader := pageread.NewPageReader(pageread.Config{
		Corpus:         corpus,
		ScrapeRequests: scrapeRequests,
		ScrapeOutcomes: scrapeOutcomes,
		Observer: pageread.PageReadObservers{
			pagereadobserversapplog.PageReadLog{},
			pagereadobserversprometheus.New(registry),
		},
		CharacterLimit:  cfg.PageFetchCharacterLimit,
		ScrapeTolerance: cfg.PageScrapeTolerance,
		FetchWait:       cfg.PageFetchWait,
	})

	toolCallServer := &http.Server{
		Handler: toolcallreceiversmcp.NewToolCallMux(toolcallreceiversmcp.Config{
			Search:              search,
			PageReader:          pageReader,
			ServiceVersion:      version,
			ToolCallConcurrency: cfg.ToolCallConcurrency,
		}),
		ReadHeaderTimeout: readHeaderLimit,
	}

	slog.InfoContext(ctx, "webresearchmcp started",
		slog.String("listen", listeners.Tools.Addr().String()),
		slog.String("webSearchEngine", cfg.WebSearchEngine),
		slog.String("corpus", cfg.CorpusMarkdownAddr),
		slog.Int("toolCallConcurrency", cfg.ToolCallConcurrency),
	)
	err = servergroup.Run(ctx, shutdownLimit, []servergroup.NamedServer{
		{Name: "tools", Listener: listeners.Tools, Server: toolCallServer},
	})
	slog.InfoContext(ctx, "webresearchmcp stopped")
	return err
}

func webSearchEngineFor(cfg ServiceConfig, registry *prometheus.Registry) websearch.SearchEngine {
	if cfg.WebSearchEngine == WebSearchEngineSearXNG {
		return searchenginessearxng.NewSearXNG(cfg.SearXNGURL, cfg.WebSearchDeadline)
	}
	return searchenginesstack.NewStack(
		[]searchenginesstack.Engine{
			{
				Name: "corpustext",
				Engine: searchenginescorpustext.NewCorpusText(
					cfg.CorpusTextURL,
					cfg.WebSearchDeadline,
				),
			},
			{
				Name:   "yacydhtsearch",
				Engine: searchenginesyacy.NewYaCy(cfg.YaCyDHTSearchURL, cfg.WebSearchDeadline),
			},
		},
		searchenginesstack.EngineObservers{
			stackobserversapplog.EngineLog{},
			stackobserversprometheus.New(registry),
		},
	)
}
