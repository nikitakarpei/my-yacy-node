package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvScrapeNATSURL           = "SCRAPE_NATS_URL"
	EnvProxyURL                = "SCRAPE_PROXY_URL"
	EnvProxyDialMode           = "SCRAPE_PROXY_DIAL_MODE"
	EnvUserAgent               = "SCRAPE_USER_AGENT"
	EnvScrapeMaxBodyBytes      = "SCRAPE_MAX_BODY_BYTES"
	EnvScrapeFetchDeadline     = "SCRAPE_FETCH_DEADLINE"
	EnvScrapeMaxRedirectHops   = "SCRAPE_MAX_REDIRECT_HOPS"
	EnvScrapeRequestDurable    = "SCRAPE_REQUEST_DURABLE"
	EnvScrapeIntakeConcurrency = "SCRAPE_INTAKE_CONCURRENCY"
	EnvScrapeRequestsInFlight  = "SCRAPE_REQUESTS_IN_FLIGHT"
	EnvScrapeRequestsKept      = "SCRAPE_REQUESTS_KEPT"
	EnvScrapeDeferralWindow    = "SCRAPE_DEFERRAL_WINDOW"
	EnvScrapePageOfferMaxBytes = "SCRAPE_PAGE_OFFER_MAX_BYTES"
	EnvScrapePageOfferMaxAge   = "SCRAPE_PAGE_OFFER_MAX_AGE"
	EnvOpsAddr                 = "PAGESCRAPE_OPS_ADDR"

	DefaultProxyDialMode           = "tunnel"
	DefaultUserAgent               = "pagescrape (+https://yacy.net)"
	DefaultScrapeMaxBodyBytes      = 2 << 20
	DefaultScrapeFetchDeadline     = 30 * time.Second
	DefaultScrapeMaxRedirectHops   = 10
	DefaultScrapeRequestDurable    = "pagescrape"
	DefaultScrapeIntakeConcurrency = 4
	DefaultScrapeRequestsInFlight  = 64
	DefaultScrapeRequestsKept      = 100_000
	DefaultScrapeDeferralWindow    = 6 * time.Hour
	DefaultScrapePageOfferMaxBytes = "1GB"
	DefaultScrapePageOfferMaxAge   = 24 * time.Hour
	DefaultOpsAddr                 = ":9090"
)

type ServiceConfig struct {
	ScrapeNATSURL           string
	ProxyURL                *url.URL
	ProxyDialMode           http.ProxyDialMode
	UserAgent               string
	MaxBodyBytes            int64
	FetchDeadline           time.Duration
	MaxRedirectHops         int
	ScrapeRequestDurable    string
	ScrapeIntakeConcurrency int
	ScrapeRequestsInFlight  int
	ScrapeRequestsKept      int64
	ScrapeDeferralWindow    time.Duration
	PageOfferMaxBytes       int64
	PageOfferMaxAge         time.Duration
	OpsAddr                 string
}

type fetchSettings struct {
	proxyURL        *url.URL
	proxyDialMode   http.ProxyDialMode
	maxBodyBytes    int64
	fetchDeadline   time.Duration
	maxRedirectHops int
}

type streamSettings struct {
	scrapeRequestsKept int64
	pageOfferMaxBytes  int64
	pageOfferMaxAge    time.Duration
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	scrapeNATSURL, err := envconfig.Required(getenv, EnvScrapeNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	fetch, err := loadFetchSettings(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	streams, err := loadStreamSettings(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	scrapeIntakeConcurrency, err := envconfig.PositiveInt(
		getenv, EnvScrapeIntakeConcurrency, DefaultScrapeIntakeConcurrency,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	scrapeRequestsInFlight, err := envconfig.PositiveInt(
		getenv, EnvScrapeRequestsInFlight, DefaultScrapeRequestsInFlight,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	scrapeDeferralWindow, err := envconfig.Duration(
		getenv, EnvScrapeDeferralWindow, DefaultScrapeDeferralWindow,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		ScrapeNATSURL:   scrapeNATSURL,
		ProxyURL:        fetch.proxyURL,
		ProxyDialMode:   fetch.proxyDialMode,
		UserAgent:       envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
		MaxBodyBytes:    fetch.maxBodyBytes,
		FetchDeadline:   fetch.fetchDeadline,
		MaxRedirectHops: fetch.maxRedirectHops,
		ScrapeRequestDurable: envconfig.String(
			getenv, EnvScrapeRequestDurable, DefaultScrapeRequestDurable,
		),
		ScrapeIntakeConcurrency: scrapeIntakeConcurrency,
		ScrapeRequestsInFlight:  scrapeRequestsInFlight,
		ScrapeRequestsKept:      streams.scrapeRequestsKept,
		ScrapeDeferralWindow:    scrapeDeferralWindow,
		PageOfferMaxBytes:       streams.pageOfferMaxBytes,
		PageOfferMaxAge:         streams.pageOfferMaxAge,
		OpsAddr:                 envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
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
		getenv, EnvScrapeMaxBodyBytes, DefaultScrapeMaxBodyBytes,
	)
	if err != nil {
		return fetchSettings{}, err
	}
	fetchDeadline, err := envconfig.Duration(
		getenv, EnvScrapeFetchDeadline, DefaultScrapeFetchDeadline,
	)
	if err != nil {
		return fetchSettings{}, err
	}
	maxRedirectHops, err := envconfig.PositiveInt(
		getenv, EnvScrapeMaxRedirectHops, DefaultScrapeMaxRedirectHops,
	)
	if err != nil {
		return fetchSettings{}, err
	}
	return fetchSettings{
		proxyURL:        proxyURL,
		proxyDialMode:   proxyDialMode,
		maxBodyBytes:    maxBodyBytes,
		fetchDeadline:   fetchDeadline,
		maxRedirectHops: maxRedirectHops,
	}, nil
}

func loadStreamSettings(getenv func(string) string) (streamSettings, error) {
	scrapeRequestsKept, err := envconfig.PositiveInt64(
		getenv, EnvScrapeRequestsKept, DefaultScrapeRequestsKept,
	)
	if err != nil {
		return streamSettings{}, err
	}
	pageOfferMaxBytes, err := envconfig.ByteSize(
		getenv, EnvScrapePageOfferMaxBytes, DefaultScrapePageOfferMaxBytes,
	)
	if err != nil {
		return streamSettings{}, err
	}
	pageOfferMaxAge, err := envconfig.Duration(
		getenv, EnvScrapePageOfferMaxAge, DefaultScrapePageOfferMaxAge,
	)
	if err != nil {
		return streamSettings{}, err
	}
	return streamSettings{
		scrapeRequestsKept: scrapeRequestsKept,
		pageOfferMaxBytes:  pageOfferMaxBytes,
		pageOfferMaxAge:    pageOfferMaxAge,
	}, nil
}
