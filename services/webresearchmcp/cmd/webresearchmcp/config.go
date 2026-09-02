package main

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvSearXNGURL                   = "SEARXNG_URL"
	EnvSearXNGSearchDeadline        = "SEARXNG_SEARCH_DEADLINE"
	EnvScrapeRequestNATSURL         = "SCRAPE_REQUEST_NATS_URL"
	EnvScrapeRequestSubject         = "SCRAPE_REQUEST_SUBJECT"
	EnvPageMarkdownNATSURL          = "PAGE_MARKDOWN_NATS_URL"
	EnvPageFetchWait                = "PAGE_FETCH_WAIT"
	EnvPageScrapeTolerance          = "PAGE_SCRAPE_TOLERANCE"
	EnvCorpusMarkdownAddr           = "CORPUSMARKDOWN_ADDR"
	EnvCorpusMarkdownRecallDeadline = "CORPUSMARKDOWN_RECALL_DEADLINE"
	EnvPageFetchCharacterLimit      = "PAGE_FETCH_CHARACTER_LIMIT"
	EnvSearchResultLimit            = "SEARCH_RESULT_LIMIT"
	EnvToolCallConcurrency          = "TOOL_CALL_CONCURRENCY"
	EnvListenAddr                   = "WEBRESEARCHMCP_LISTEN_ADDR"
	EnvOpsAddr                      = "WEBRESEARCHMCP_OPS_ADDR"

	DefaultSearXNGSearchDeadline        = 10 * time.Second
	DefaultPageFetchWait                = 10 * time.Second
	DefaultPageScrapeTolerance          = time.Hour
	DefaultCorpusMarkdownRecallDeadline = 5 * time.Second
	DefaultPageFetchCharacterLimit      = 5000
	DefaultSearchResultLimit            = 10
	DefaultToolCallConcurrency          = 8
	DefaultListenAddr                   = ":8095"
	DefaultOpsAddr                      = ":9090"
)

var DefaultScrapeRequestSubject = pagescrapecontract.ScrapeRequestSubject

type ServiceConfig struct {
	SearXNGURL                   string
	SearXNGSearchDeadline        time.Duration
	ScrapeRequestNATSURL         string
	ScrapeRequestSubject         string
	PageMarkdownNATSURL          string
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
	searxngSearchDeadline        time.Duration
	pageFetchWait                time.Duration
	pageScrapeTolerance          time.Duration
	corpusMarkdownRecallDeadline time.Duration
}

func loadCallDeadlines(getenv func(string) string) (callDeadlines, error) {
	searxngSearchDeadline, err := envconfig.Duration(
		getenv,
		EnvSearXNGSearchDeadline,
		DefaultSearXNGSearchDeadline,
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
		searxngSearchDeadline:        searxngSearchDeadline,
		pageFetchWait:                pageFetchWait,
		pageScrapeTolerance:          pageScrapeTolerance,
		corpusMarkdownRecallDeadline: corpusMarkdownRecallDeadline,
	}, nil
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	searxngURL, err := envconfig.RequiredHTTPURL(getenv, EnvSearXNGURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	scrapeRequestNATSURL, err := envconfig.Required(getenv, EnvScrapeRequestNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageMarkdownNATSURL, err := envconfig.Required(getenv, EnvPageMarkdownNATSURL)
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
		SearXNGURL:            searxngURL.String(),
		SearXNGSearchDeadline: deadlines.searxngSearchDeadline,
		ScrapeRequestNATSURL:  scrapeRequestNATSURL,
		ScrapeRequestSubject: envconfig.String(
			getenv,
			EnvScrapeRequestSubject,
			DefaultScrapeRequestSubject,
		),
		PageMarkdownNATSURL:          pageMarkdownNATSURL,
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
