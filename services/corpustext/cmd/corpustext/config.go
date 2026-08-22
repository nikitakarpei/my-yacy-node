package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvScrapeRequestNATSURL     = "SCRAPE_REQUEST_NATS_URL"
	EnvNATSScrapeRequestSubject = "NATS_SCRAPE_REQUEST_SUBJECT"
	EnvNATSScrapeRequestDurable = "NATS_SCRAPE_REQUEST_DURABLE"
	EnvProxyURL                 = "CORPUSTEXT_PROXY_URL"
	EnvProxyDialMode            = "CORPUSTEXT_PROXY_DIAL_MODE"
	EnvUserAgent                = "CORPUSTEXT_USER_AGENT"
	EnvMaxBodyBytes             = "CORPUSTEXT_MAX_BODY_BYTES"
	EnvFetchDeadline            = "CORPUSTEXT_FETCH_DEADLINE"
	EnvConcurrency              = "CORPUSTEXT_CONCURRENCY"
	EnvSearchIndexEngine        = "SEARCH_INDEX_ENGINE"
	EnvElasticsearchURL         = "ELASTICSEARCH_URL"
	EnvElasticsearchIndex       = "ELASTICSEARCH_INDEX"
	EnvManticoreURL             = "MANTICORE_URL"
	EnvManticoreTable           = "MANTICORE_TABLE"
	EnvLanguages                = "CORPUSTEXT_LANGUAGES"
	EnvOpsAddr                  = "CORPUSTEXT_OPS_ADDR"

	DefaultOpsAddr              = ":9090"
	DefaultScrapeRequestDurable = "corpustext"
	DefaultProxyDialMode        = "tunnel"
	DefaultUserAgent            = "corpustext (+https://yacy.net)"
	DefaultMaxBodyBytes         = 2 << 20
	DefaultFetchDeadline        = 30 * time.Second
	DefaultConcurrency          = 4
	DefaultIndexBaseName        = "yacy_text"

	SearchIndexEngineElasticsearch = "elasticsearch"
	SearchIndexEngineManticore     = "manticore"
)

var DefaultScrapeRequestSubject = scraperequestcontract.ScrapeRequestSubject

type ServiceConfig struct {
	ScrapeRequestNATSURL string
	ScrapeRequestSubject string
	ScrapeRequestDurable string
	ProxyURL             *url.URL
	ProxyDialMode        http.ProxyDialMode
	UserAgent            string
	MaxBodyBytes         int64
	FetchDeadline        time.Duration
	Concurrency          int
	SearchIndexEngine    string
	ElasticsearchURL     string
	ElasticsearchIndex   string
	ManticoreURL         string
	ManticoreTable       string
	Languages            []string
	OpsAddr              string
}

type fetchSettings struct {
	proxyURL      *url.URL
	proxyDialMode http.ProxyDialMode
	maxBodyBytes  int64
	fetchDeadline time.Duration
}

func loadFetchSettings(getenv func(string) string) (fetchSettings, error) {
	proxyURL, err := envconfig.RequiredHTTPURL(getenv, EnvProxyURL)
	if err != nil {
		return fetchSettings{}, err
	}
	proxyDialMode, err := http.ProxyDialModeNamed(
		envconfig.String(getenv, EnvProxyDialMode, DefaultProxyDialMode),
	)
	if err != nil {
		return fetchSettings{}, fmt.Errorf("%s: %w", EnvProxyDialMode, err)
	}
	maxBodyBytes, err := envconfig.PositiveInt64(getenv, EnvMaxBodyBytes, DefaultMaxBodyBytes)
	if err != nil {
		return fetchSettings{}, err
	}
	fetchDeadline, err := envconfig.Duration(getenv, EnvFetchDeadline, DefaultFetchDeadline)
	if err != nil {
		return fetchSettings{}, err
	}
	return fetchSettings{
		proxyURL:      proxyURL,
		proxyDialMode: proxyDialMode,
		maxBodyBytes:  maxBodyBytes,
		fetchDeadline: fetchDeadline,
	}, nil
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	scrapeRequestNATSURL, err := envconfig.Required(getenv, EnvScrapeRequestNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	fetch, err := loadFetchSettings(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}

	concurrency, err := envconfig.PositiveInt(getenv, EnvConcurrency, DefaultConcurrency)
	if err != nil {
		return ServiceConfig{}, err
	}

	cfg := ServiceConfig{
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		ScrapeRequestSubject: envconfig.String(
			getenv,
			EnvNATSScrapeRequestSubject,
			DefaultScrapeRequestSubject,
		),
		ScrapeRequestDurable: envconfig.String(
			getenv,
			EnvNATSScrapeRequestDurable,
			DefaultScrapeRequestDurable,
		),
		ProxyURL:          fetch.proxyURL,
		ProxyDialMode:     fetch.proxyDialMode,
		UserAgent:         envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
		MaxBodyBytes:      fetch.maxBodyBytes,
		FetchDeadline:     fetch.fetchDeadline,
		Concurrency:       concurrency,
		SearchIndexEngine: strings.TrimSpace(getenv(EnvSearchIndexEngine)),
		Languages:         envconfig.List(getenv, EnvLanguages),
		OpsAddr:           envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}
	if cfg.SearchIndexEngine == "" {
		return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvSearchIndexEngine)
	}

	switch cfg.SearchIndexEngine {
	case SearchIndexEngineElasticsearch:
		cfg.ElasticsearchURL = strings.TrimSpace(getenv(EnvElasticsearchURL))
		if cfg.ElasticsearchURL == "" {
			return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvElasticsearchURL)
		}
		cfg.ElasticsearchIndex = envconfig.String(
			getenv,
			EnvElasticsearchIndex,
			DefaultIndexBaseName,
		)
	case SearchIndexEngineManticore:
		cfg.ManticoreURL = strings.TrimSpace(getenv(EnvManticoreURL))
		if cfg.ManticoreURL == "" {
			return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvManticoreURL)
		}
		cfg.ManticoreTable = envconfig.String(getenv, EnvManticoreTable, DefaultIndexBaseName)
	default:
		return ServiceConfig{}, unknownSearchIndexEngine(cfg.SearchIndexEngine)
	}

	return cfg, nil
}

func unknownSearchIndexEngine(engine string) error {
	return fmt.Errorf("%s: unknown engine %q", EnvSearchIndexEngine, engine)
}
