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
	EnvScrapeProxyURL           = "SCRAPE_PROXY_URL"
	EnvScrapeProxyDialMode      = "SCRAPE_PROXY_DIAL_MODE"
	EnvScrapeUserAgent          = "SCRAPE_USER_AGENT"
	EnvScrapeMaxBodyBytes       = "YACY_SCRAPE_MAX_BODY_BYTES"
	EnvScrapeFetchDeadline      = "YACY_SCRAPE_FETCH_DEADLINE"
	EnvScrapeConcurrency        = "YACY_SCRAPE_CONCURRENCY"

	DefaultScrapeRequestDurable = "yacy-node"
	DefaultScrapeProxyDialMode  = "tunnel"
	DefaultScrapeUserAgent      = "yacy-rwi-node (+https://yacy.net)"
	DefaultScrapeMaxBodyBytes   = 2 << 20
	DefaultScrapeFetchDeadline  = 30 * time.Second
	DefaultScrapeConcurrency    = 4
)

var DefaultScrapeRequestSubject = scraperequestcontract.ScrapeRequestSubject

type ScrapeRequestIngestConfig struct {
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

func (c ScrapeRequestIngestConfig) Enabled() bool {
	return c.ScrapeRequestNATSURL != ""
}

func loadScrapeRequestIngestConfig(getenv func(string) string) (ScrapeRequestIngestConfig, error) {
	scrapeRequestNATSURL := strings.TrimSpace(getenv(EnvScrapeRequestNATSURL))
	if scrapeRequestNATSURL == "" {
		return ScrapeRequestIngestConfig{}, nil
	}
	proxyURL, err := envconfig.RequiredHTTPURL(getenv, EnvScrapeProxyURL)
	if err != nil {
		return ScrapeRequestIngestConfig{}, err
	}
	proxyDialMode, err := http.ProxyDialModeNamed(
		envconfig.String(getenv, EnvScrapeProxyDialMode, DefaultScrapeProxyDialMode),
	)
	if err != nil {
		return ScrapeRequestIngestConfig{}, fmt.Errorf("%s: %w", EnvScrapeProxyDialMode, err)
	}
	maxBodyBytes, err := envconfig.PositiveInt64(
		getenv, EnvScrapeMaxBodyBytes, DefaultScrapeMaxBodyBytes,
	)
	if err != nil {
		return ScrapeRequestIngestConfig{}, err
	}
	fetchDeadline, err := envconfig.Duration(
		getenv, EnvScrapeFetchDeadline, DefaultScrapeFetchDeadline,
	)
	if err != nil {
		return ScrapeRequestIngestConfig{}, err
	}
	concurrency, err := envconfig.PositiveInt(
		getenv, EnvScrapeConcurrency, DefaultScrapeConcurrency,
	)
	if err != nil {
		return ScrapeRequestIngestConfig{}, err
	}

	return ScrapeRequestIngestConfig{
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		ScrapeRequestSubject: envconfig.String(
			getenv, EnvNATSScrapeRequestSubject, DefaultScrapeRequestSubject,
		),
		ScrapeRequestDurable: envconfig.String(
			getenv, EnvNATSScrapeRequestDurable, DefaultScrapeRequestDurable,
		),
		ProxyURL:      proxyURL,
		ProxyDialMode: proxyDialMode,
		UserAgent:     envconfig.String(getenv, EnvScrapeUserAgent, DefaultScrapeUserAgent),
		MaxBodyBytes:  maxBodyBytes,
		FetchDeadline: fetchDeadline,
		Concurrency:   concurrency,
	}, nil
}
