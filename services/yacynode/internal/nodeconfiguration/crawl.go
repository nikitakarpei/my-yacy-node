package nodeconfiguration

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
	EnvCrawlProxyURL            = "YACY_CRAWL_PROXY_URL"
	EnvCrawlProxyDialMode       = "YACY_CRAWL_PROXY_DIAL_MODE"
	EnvCrawlUserAgent           = "YACY_CRAWL_USER_AGENT"
	EnvCrawlMaxBodyBytes        = "YACY_CRAWL_MAX_BODY_BYTES"
	EnvCrawlFetchDeadline       = "YACY_CRAWL_FETCH_DEADLINE"
	EnvCrawlConcurrency         = "YACY_CRAWL_CONCURRENCY"

	DefaultScrapeRequestDurable = "yacy-node"
	DefaultCrawlProxyDialMode   = "tunnel"
	DefaultCrawlUserAgent       = "yacy-rwi-node (+https://yacy.net)"
	DefaultCrawlMaxBodyBytes    = 2 << 20
	DefaultCrawlFetchDeadline   = 30 * time.Second
	DefaultCrawlConcurrency     = 4
)

var DefaultScrapeRequestSubject = scraperequestcontract.ScrapeRequestSubject

type CrawlConfig struct {
	ScrapeRequestNATSURL string
	ScrapeRequestSubject string
	ScrapeRequestDurable string
	ProxyURL             *url.URL
	ProxyDialMode        http.ProxyDialMode
	UserAgent            string
	MaxBodyBytes         int64
	FetchDeadline        time.Duration
	Concurrency          int
}

func (c CrawlConfig) Enabled() bool {
	return c.ScrapeRequestNATSURL != ""
}

func loadCrawlConfig(getenv func(string) string) (CrawlConfig, error) {
	scrapeRequestNATSURL := strings.TrimSpace(getenv(EnvScrapeRequestNATSURL))
	if scrapeRequestNATSURL == "" {
		return CrawlConfig{}, nil
	}
	proxyURL, err := envconfig.RequiredHTTPURL(getenv, EnvCrawlProxyURL)
	if err != nil {
		return CrawlConfig{}, err
	}
	proxyDialMode, err := http.ProxyDialModeNamed(
		envconfig.String(getenv, EnvCrawlProxyDialMode, DefaultCrawlProxyDialMode),
	)
	if err != nil {
		return CrawlConfig{}, fmt.Errorf("%s: %w", EnvCrawlProxyDialMode, err)
	}
	maxBodyBytes, err := envconfig.PositiveInt64(
		getenv, EnvCrawlMaxBodyBytes, DefaultCrawlMaxBodyBytes,
	)
	if err != nil {
		return CrawlConfig{}, err
	}
	fetchDeadline, err := envconfig.Duration(
		getenv, EnvCrawlFetchDeadline, DefaultCrawlFetchDeadline,
	)
	if err != nil {
		return CrawlConfig{}, err
	}
	concurrency, err := envconfig.PositiveInt(
		getenv, EnvCrawlConcurrency, DefaultCrawlConcurrency,
	)
	if err != nil {
		return CrawlConfig{}, err
	}

	return CrawlConfig{
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		ScrapeRequestSubject: envconfig.String(
			getenv, EnvNATSScrapeRequestSubject, DefaultScrapeRequestSubject,
		),
		ScrapeRequestDurable: envconfig.String(
			getenv, EnvNATSScrapeRequestDurable, DefaultScrapeRequestDurable,
		),
		ProxyURL:      proxyURL,
		ProxyDialMode: proxyDialMode,
		UserAgent:     envconfig.String(getenv, EnvCrawlUserAgent, DefaultCrawlUserAgent),
		MaxBodyBytes:  maxBodyBytes,
		FetchDeadline: fetchDeadline,
		Concurrency:   concurrency,
	}, nil
}
