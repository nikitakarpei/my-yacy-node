package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvCrawlNATSURL             = "CRAWL_NATS_URL"
	EnvPageMarkdownNATSURL      = "PAGE_MARKDOWN_NATS_URL"
	EnvNATSScrapeRequestSubject = "NATS_SCRAPE_REQUEST_SUBJECT"
	EnvNATSScrapeRequestDurable = "NATS_SCRAPE_REQUEST_DURABLE"
	EnvProxyURL                 = "CORPUSMARKDOWN_PROXY_URL"
	EnvProxyDialMode            = "CORPUSMARKDOWN_PROXY_DIAL_MODE"
	EnvUserAgent                = "CORPUSMARKDOWN_USER_AGENT"
	EnvMaxBodyBytes             = "CORPUSMARKDOWN_MAX_BODY_BYTES"
	EnvFetchDeadline            = "CORPUSMARKDOWN_FETCH_DEADLINE"
	EnvConcurrency              = "CORPUSMARKDOWN_CONCURRENCY"
	EnvListenAddr               = "CORPUSMARKDOWN_LISTEN_ADDR"
	EnvOpsAddr                  = "CORPUSMARKDOWN_OPS_ADDR"

	DefaultListenAddr           = ":8094"
	DefaultOpsAddr              = ":9090"
	DefaultScrapeRequestDurable = "corpusmarkdown"
	DefaultProxyDialMode        = "tunnel"
	DefaultUserAgent            = "corpusmarkdown (+https://yacy.net)"
	DefaultMaxBodyBytes         = 2 << 20
	DefaultFetchDeadline        = 30 * time.Second
	DefaultConcurrency          = 4
)

var DefaultScrapeRequestSubject = scraperequestcontract.ScrapeRequestSubject

type ServiceConfig struct {
	CrawlNATSURL         string
	PageMarkdownNATSURL  string
	ScrapeRequestSubject string
	ScrapeRequestDurable string
	ProxyURL             *url.URL
	ProxyDialMode        http.ProxyDialMode
	UserAgent            string
	MaxBodyBytes         int64
	FetchDeadline        time.Duration
	Concurrency          int
	ListenAddr           string
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
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
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
	concurrency, err := envconfig.PositiveInt(getenv, EnvConcurrency, DefaultConcurrency)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		CrawlNATSURL:        crawlNATSURL,
		PageMarkdownNATSURL: pageMarkdownNATSURL,
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
		ProxyURL:      fetch.proxyURL,
		ProxyDialMode: fetch.proxyDialMode,
		UserAgent:     envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
		MaxBodyBytes:  fetch.maxBodyBytes,
		FetchDeadline: fetch.fetchDeadline,
		Concurrency:   concurrency,
		ListenAddr:    envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:       envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
