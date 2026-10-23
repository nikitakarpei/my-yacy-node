package webresearchmcpassembly

import (
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvWebSearchEngine              = "WEB_SEARCH_ENGINE"
	EnvWebSearchDeadline            = "WEB_SEARCH_DEADLINE"
	EnvSearXNGURL                   = "SEARXNG_URL"
	EnvYaCyDHTSearchURL             = "YACYDHTSEARCH_URL"
	EnvCorpusTextURL                = "CORPUSTEXT_URL"
	EnvScrapeRequestNATSURL         = "SCRAPE_REQUEST_NATS_URL"
	EnvPageFetchWait                = "PAGE_FETCH_WAIT"
	EnvPageScrapeTolerance          = "PAGE_SCRAPE_TOLERANCE"
	EnvCorpusMarkdownAddr           = "CORPUSMARKDOWN_ADDR"
	EnvCorpusMarkdownRecallDeadline = "CORPUSMARKDOWN_RECALL_DEADLINE"
	EnvPageFetchCharacterLimit      = "PAGE_FETCH_CHARACTER_LIMIT"
	EnvSearchResultLimit            = "SEARCH_RESULT_LIMIT"
	EnvToolCallConcurrency          = "TOOL_CALL_CONCURRENCY"
	EnvListenAddr                   = "WEBRESEARCHMCP_LISTEN_ADDR"
	EnvOpsAddr                      = "WEBRESEARCHMCP_OPS_ADDR"

	DefaultWebSearchEngine              = WebSearchEngineSearXNG
	DefaultWebSearchDeadline            = 10 * time.Second
	DefaultPageFetchWait                = 10 * time.Second
	DefaultPageScrapeTolerance          = time.Hour
	DefaultCorpusMarkdownRecallDeadline = 5 * time.Second
	DefaultPageFetchCharacterLimit      = 5000
	DefaultSearchResultLimit            = 10
	DefaultToolCallConcurrency          = 8
	DefaultListenAddr                   = ":8095"
	DefaultOpsAddr                      = ":9090"

	WebSearchEngineSearXNG = "searxng"
	WebSearchEngineStack   = "stack"
)

type ServiceConfig struct {
	WebSearchEngine              string
	WebSearchDeadline            time.Duration
	SearXNGURL                   string
	YaCyDHTSearchURL             string
	CorpusTextURL                string
	ScrapeRequestNATSURL         string
	PageFetchWait                time.Duration
	PageScrapeTolerance          time.Duration
	CorpusMarkdownAddr           string
	CorpusMarkdownRecallDeadline time.Duration
	PageFetchCharacterLimit      int
	SearchResultLimit            int
	ToolCallConcurrency          int
	ListenAddr                   string
	OpsAddr                      string
}

type answerLimits struct {
	pageFetchCharacterLimit int
	searchResultLimit       int
	toolCallConcurrency     int
}

func loadAnswerLimits(getenv func(string) string) (answerLimits, error) {
	pageFetchCharacterLimit, err := envconfig.PositiveInt(
		getenv,
		EnvPageFetchCharacterLimit,
		DefaultPageFetchCharacterLimit,
	)
	if err != nil {
		return answerLimits{}, err
	}
	searchResultLimit, err := envconfig.PositiveInt(
		getenv,
		EnvSearchResultLimit,
		DefaultSearchResultLimit,
	)
	if err != nil {
		return answerLimits{}, err
	}
	toolCallConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvToolCallConcurrency,
		DefaultToolCallConcurrency,
	)
	if err != nil {
		return answerLimits{}, err
	}
	return answerLimits{
		pageFetchCharacterLimit: pageFetchCharacterLimit,
		searchResultLimit:       searchResultLimit,
		toolCallConcurrency:     toolCallConcurrency,
	}, nil
}

type callDeadlines struct {
	webSearchDeadline            time.Duration
	pageFetchWait                time.Duration
	pageScrapeTolerance          time.Duration
	corpusMarkdownRecallDeadline time.Duration
}

func loadCallDeadlines(getenv func(string) string) (callDeadlines, error) {
	webSearchDeadline, err := envconfig.Duration(
		getenv,
		EnvWebSearchDeadline,
		DefaultWebSearchDeadline,
	)
	if err != nil {
		return callDeadlines{}, err
	}
	pageFetchWait, err := envconfig.Duration(getenv, EnvPageFetchWait, DefaultPageFetchWait)
	if err != nil {
		return callDeadlines{}, err
	}
	pageScrapeTolerance, err := envconfig.NonNegativeDuration(
		getenv,
		EnvPageScrapeTolerance,
		DefaultPageScrapeTolerance,
	)
	if err != nil {
		return callDeadlines{}, err
	}
	corpusMarkdownRecallDeadline, err := envconfig.Duration(
		getenv,
		EnvCorpusMarkdownRecallDeadline,
		DefaultCorpusMarkdownRecallDeadline,
	)
	if err != nil {
		return callDeadlines{}, err
	}
	return callDeadlines{
		webSearchDeadline:            webSearchDeadline,
		pageFetchWait:                pageFetchWait,
		pageScrapeTolerance:          pageScrapeTolerance,
		corpusMarkdownRecallDeadline: corpusMarkdownRecallDeadline,
	}, nil
}

type webSearchEngines struct {
	engine           string
	searxngURL       string
	yacyDHTSearchURL string
	corpusTextURL    string
}

func loadWebSearchEngines(getenv func(string) string) (webSearchEngines, error) {
	engines := webSearchEngines{
		engine: envconfig.String(getenv, EnvWebSearchEngine, DefaultWebSearchEngine),
	}
	switch engines.engine {
	case WebSearchEngineSearXNG:
		searxngURL, err := envconfig.RequiredHTTPURL(getenv, EnvSearXNGURL)
		if err != nil {
			return webSearchEngines{}, err
		}
		engines.searxngURL = searxngURL.String()
	case WebSearchEngineStack:
		yacyDHTSearchURL, yacyDHTSearchErr := envconfig.RequiredHTTPURL(getenv, EnvYaCyDHTSearchURL)
		corpusTextURL, corpusTextErr := envconfig.RequiredHTTPURL(getenv, EnvCorpusTextURL)
		if err := errors.Join(yacyDHTSearchErr, corpusTextErr); err != nil {
			return webSearchEngines{}, err
		}
		engines.yacyDHTSearchURL = yacyDHTSearchURL.String()
		engines.corpusTextURL = corpusTextURL.String()
	default:
		return webSearchEngines{}, fmt.Errorf(
			"%s: unknown engine %q", EnvWebSearchEngine, engines.engine,
		)
	}
	return engines, nil
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	engines, err := loadWebSearchEngines(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	scrapeRequestNATSURL, err := envconfig.Required(getenv, EnvScrapeRequestNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	corpusMarkdownAddr, err := envconfig.Required(getenv, EnvCorpusMarkdownAddr)
	if err != nil {
		return ServiceConfig{}, err
	}
	deadlines, err := loadCallDeadlines(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	limits, err := loadAnswerLimits(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		WebSearchEngine:              engines.engine,
		WebSearchDeadline:            deadlines.webSearchDeadline,
		SearXNGURL:                   engines.searxngURL,
		YaCyDHTSearchURL:             engines.yacyDHTSearchURL,
		CorpusTextURL:                engines.corpusTextURL,
		ScrapeRequestNATSURL:         scrapeRequestNATSURL,
		PageFetchWait:                deadlines.pageFetchWait,
		PageScrapeTolerance:          deadlines.pageScrapeTolerance,
		CorpusMarkdownAddr:           corpusMarkdownAddr,
		CorpusMarkdownRecallDeadline: deadlines.corpusMarkdownRecallDeadline,
		PageFetchCharacterLimit:      limits.pageFetchCharacterLimit,
		SearchResultLimit:            limits.searchResultLimit,
		ToolCallConcurrency:          limits.toolCallConcurrency,
		ListenAddr:                   envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:                      envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
