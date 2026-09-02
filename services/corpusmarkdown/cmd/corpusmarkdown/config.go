package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvScrapeRequestNATSURL           = "SCRAPE_REQUEST_NATS_URL"
	EnvPageMarkdownNATSURL            = "PAGE_MARKDOWN_NATS_URL"
	EnvScrapeRequestSubject           = "SCRAPE_REQUEST_SUBJECT"
	EnvScrapeRequestDurable           = "SCRAPE_REQUEST_DURABLE"
	EnvProxyURL                       = "SCRAPE_PROXY_URL"
	EnvProxyDialMode                  = "SCRAPE_PROXY_DIAL_MODE"
	EnvUserAgent                      = "SCRAPE_USER_AGENT"
	EnvScrapeMaxBodyBytes             = "SCRAPE_MAX_BODY_BYTES"
	EnvScrapeFetchDeadline            = "SCRAPE_FETCH_DEADLINE"
	EnvScrapeRequestIntakeConcurrency = "SCRAPE_REQUEST_INTAKE_CONCURRENCY"
	EnvListenAddr                     = "CORPUSMARKDOWN_LISTEN_ADDR"
	EnvOpsAddr                        = "CORPUSMARKDOWN_OPS_ADDR"

	DefaultListenAddr                     = ":8094"
	DefaultOpsAddr                        = ":9090"
	DefaultScrapeRequestDurable           = "corpusmarkdown"
	DefaultProxyDialMode                  = "tunnel"
	DefaultUserAgent                      = "corpusmarkdown (+https://yacy.net)"
	DefaultScrapeMaxBodyBytes             = 2 << 20
	DefaultScrapeFetchDeadline            = 30 * time.Second
	DefaultScrapeRequestIntakeConcurrency = 4
)

var DefaultScrapeRequestSubject = pagescrapecontract.ScrapeRequestSubject

type ServiceConfig struct {
	ScrapeRequestNATSURL           string
	PageMarkdownNATSURL            string
	ScrapeRequestSubject           string
	ScrapeRequestDurable           string
	ProxyURL                       *url.URL
	ProxyDialMode                  http.ProxyDialMode
	UserAgent                      string
	MaxBodyBytes                   int64
	FetchDeadline                  time.Duration
	ScrapeRequestIntakeConcurrency int
	ListenAddr                     string
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
	pageMarkdownNATSURL, err := envconfig.Required(getenv, EnvPageMarkdownNATSURL)
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

	return ServiceConfig{
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		PageMarkdownNATSURL:  pageMarkdownNATSURL,
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
		ListenAddr:                     envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:                        envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
