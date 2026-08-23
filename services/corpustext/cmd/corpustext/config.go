package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvScrapeRequestNATSURL           = "SCRAPE_REQUEST_NATS_URL"
	EnvScrapeRequestSubject           = "SCRAPE_REQUEST_SUBJECT"
	EnvScrapeRequestDurable           = "SCRAPE_REQUEST_DURABLE"
	EnvProxyURL                       = "SCRAPE_PROXY_URL"
	EnvProxyDialMode                  = "SCRAPE_PROXY_DIAL_MODE"
	EnvUserAgent                      = "SCRAPE_USER_AGENT"
	EnvScrapeMaxBodyBytes             = "SCRAPE_MAX_BODY_BYTES"
	EnvScrapeFetchDeadline            = "SCRAPE_FETCH_DEADLINE"
	EnvScrapeRequestIntakeConcurrency = "SCRAPE_REQUEST_INTAKE_CONCURRENCY"
	EnvSearchIndexEngine              = "SEARCH_INDEX_ENGINE"
	EnvElasticsearchURL               = "ELASTICSEARCH_URL"
	EnvElasticsearchIndex             = "ELASTICSEARCH_INDEX"
	EnvManticoreURL                   = "MANTICORE_URL"
	EnvManticoreTable                 = "MANTICORE_TABLE"
	EnvLanguages                      = "CORPUSTEXT_LANGUAGES"
	EnvOpsAddr                        = "CORPUSTEXT_OPS_ADDR"

	DefaultOpsAddr                        = ":9090"
	DefaultScrapeRequestDurable           = "corpustext"
	DefaultProxyDialMode                  = "tunnel"
	DefaultUserAgent                      = "corpustext (+https://yacy.net)"
	DefaultScrapeMaxBodyBytes             = 2 << 20
	DefaultScrapeFetchDeadline            = 30 * time.Second
	DefaultScrapeRequestIntakeConcurrency = 4
	DefaultIndexBaseName                  = "yacy_text"

	SearchIndexEngineElasticsearch = "elasticsearch"
	SearchIndexEngineManticore     = "manticore"
)

var DefaultScrapeRequestSubject = scraperequestcontract.ScrapeRequestSubject

type ServiceConfig struct {
	ScrapeRequestNATSURL           string
	ScrapeRequestSubject           string
	ScrapeRequestDurable           string
	ProxyURL                       *url.URL
	ProxyDialMode                  http.ProxyDialMode
	UserAgent                      string
	MaxBodyBytes                   int64
	FetchDeadline                  time.Duration
	ScrapeRequestIntakeConcurrency int
	SearchIndexEngine              string
	ElasticsearchURL               string
	ElasticsearchIndex             string
	ManticoreURL                   string
	ManticoreTable                 string
	Languages                      []string
	OpsAddr                        string
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
	maxBodyBytes, err := envconfig.PositiveInt64(
		getenv,
		EnvScrapeMaxBodyBytes,
		DefaultScrapeMaxBodyBytes,
	)
	if err != nil {
		return fetchSettings{}, err
	}
	fetchDeadline, err := envconfig.Duration(
		getenv,
		EnvScrapeFetchDeadline,
		DefaultScrapeFetchDeadline,
	)
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

	scrapeRequestIntakeConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvScrapeRequestIntakeConcurrency,
		DefaultScrapeRequestIntakeConcurrency,
	)
	if err != nil {
		return ServiceConfig{}, err
	}

	cfg := ServiceConfig{
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		ScrapeRequestSubject: envconfig.String(
			getenv,
			EnvScrapeRequestSubject,
			DefaultScrapeRequestSubject,
		),
		ScrapeRequestDurable: envconfig.String(
			getenv,
			EnvScrapeRequestDurable,
			DefaultScrapeRequestDurable,
		),
		ProxyURL:                       fetch.proxyURL,
		ProxyDialMode:                  fetch.proxyDialMode,
		UserAgent:                      envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
		MaxBodyBytes:                   fetch.maxBodyBytes,
		FetchDeadline:                  fetch.fetchDeadline,
		ScrapeRequestIntakeConcurrency: scrapeRequestIntakeConcurrency,
		SearchIndexEngine:              strings.TrimSpace(getenv(EnvSearchIndexEngine)),
		Languages:                      envconfig.List(getenv, EnvLanguages),
		OpsAddr:                        envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
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
