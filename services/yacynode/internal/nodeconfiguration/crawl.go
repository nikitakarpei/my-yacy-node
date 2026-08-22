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
	EnvCrawlNATSURL             = "CRAWL_NATS_URL"
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
	NATSURL              string
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
	return c.NATSURL != ""
}

func loadCrawlConfig(getenv func(string) string) (CrawlConfig, error) {
	natsURL := strings.TrimSpace(getenv(EnvCrawlNATSURL))
	if natsURL == "" {
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
		NATSURL: natsURL,
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
