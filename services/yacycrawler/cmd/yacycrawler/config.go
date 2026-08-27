package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	acceptedordersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/acceptedorders/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
	visitclaimsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitclaims/jetstream"
)

const (
	EnvCrawlNATSURL         = "CRAWL_NATS_URL"
	EnvScrapeRequestNATSURL = "SCRAPE_REQUEST_NATS_URL"
	EnvCrawlOrdersSubject   = "CRAWL_ORDERS_SUBJECT"
	EnvCrawlOrdersDurable   = "CRAWL_ORDERS_DURABLE"
	EnvPendingVisitDurable  = "PENDING_VISIT_DURABLE"

	EnvProxyURL         = "YACYCRAWLER_FETCH_PROXY_URL"
	EnvProxyDialMode    = "YACYCRAWLER_FETCH_PROXY_DIAL_MODE"
	EnvFetchConcurrency = "YACYCRAWLER_FETCH_CONCURRENCY"

	EnvMaxBodyBytes  = "YACYCRAWLER_MAX_BODY_BYTES"
	EnvFetchDeadline = "YACYCRAWLER_FETCH_DEADLINE"
	EnvOpsAddr       = "YACYCRAWLER_OPS_ADDR"
	EnvUserAgent     = "YACYCRAWLER_FETCH_USER_AGENT"
	EnvRecrawlGrace  = "YACYCRAWLER_RECRAWL_GRACE"

	DefaultCrawlOrdersSubject  = "yacy.crawl.orders"
	DefaultCrawlOrdersDurable  = "yacycrawler"
	DefaultPendingVisitDurable = "yacycrawler-visits"
	DefaultFetchConcurrency    = 4
	DefaultMaxBodyBytes        = 2 << 20
	DefaultFetchDeadline       = 30 * time.Second
	DefaultOpsAddr             = ":9090"
	DefaultUserAgent           = "yacycrawler (+https://yacy.net)"
	DefaultProxyDialMode       = "tunnel"

	DefaultRecrawlGrace       = time.Hour
	DefaultPageVisitRetention = 30 * 24 * time.Hour
	DefaultPageVisitMaxBytes  = 256 << 20

	DefaultVisitClaimRetention    = 7 * 24 * time.Hour
	DefaultVisitClaimMaxBytes     = 1 << 30
	DefaultAcceptedOrderRetention = 7 * 24 * time.Hour
	DefaultAcceptedOrderMaxBytes  = 64 << 20
)

type ServiceConfig struct {
	CrawlNATSURL         string
	ScrapeRequestNATSURL string
	CrawlOrdersSubject   string
	CrawlOrdersDurable   string
	PendingVisitDurable  string
	ProxyURL             *url.URL
	ProxyDialMode        http.ProxyDialMode
	FetchConcurrency     int
	MaxBodyBytes         int64
	FetchDeadline        time.Duration
	OpsAddr              string
	UserAgent            string
	RecrawlGrace         time.Duration
}

func (ServiceConfig) PageVisitBucketSpec() dueaftergrace.BucketSpec {
	return dueaftergrace.BucketSpec{
		MaxBytes:  DefaultPageVisitMaxBytes,
		Retention: DefaultPageVisitRetention,
	}
}

func (ServiceConfig) VisitClaimBucketSpec() visitclaimsjetstream.BucketSpec {
	return visitclaimsjetstream.BucketSpec{
		MaxBytes:  DefaultVisitClaimMaxBytes,
		Retention: DefaultVisitClaimRetention,
	}
}

func (ServiceConfig) AcceptedOrderBucketSpec() acceptedordersjetstream.BucketSpec {
	return acceptedordersjetstream.BucketSpec{
		MaxBytes:  DefaultAcceptedOrderMaxBytes,
		Retention: DefaultAcceptedOrderRetention,
	}
}

type serviceLimits struct {
	fetchConcurrency int
	maxBodyBytes     int64
	fetchDeadline    time.Duration
}

func loadServiceLimits(getenv func(string) string) (serviceLimits, error) {
	fetchConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvFetchConcurrency,
		DefaultFetchConcurrency,
	)
	if err != nil {
		return serviceLimits{}, err
	}
	maxBodyBytes, err := envconfig.PositiveInt64(getenv, EnvMaxBodyBytes, DefaultMaxBodyBytes)
	if err != nil {
		return serviceLimits{}, err
	}
	fetchDeadline, err := envconfig.Duration(getenv, EnvFetchDeadline, DefaultFetchDeadline)
	if err != nil {
		return serviceLimits{}, err
	}

	return serviceLimits{
		fetchConcurrency: fetchConcurrency,
		maxBodyBytes:     maxBodyBytes,
		fetchDeadline:    fetchDeadline,
	}, nil
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	scrapeRequestNATSURL, err := envconfig.Required(getenv, EnvScrapeRequestNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	proxyURL, err := envconfig.RequiredHTTPURL(getenv, EnvProxyURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	proxyDialMode, err := proxyDialModeFromEnv(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	limits, err := loadServiceLimits(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	recrawlGrace, err := envconfig.NonNegativeDuration(getenv, EnvRecrawlGrace, DefaultRecrawlGrace)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		CrawlNATSURL:         crawlNATSURL,
		ScrapeRequestNATSURL: scrapeRequestNATSURL,
		CrawlOrdersSubject: envconfig.String(
			getenv,
			EnvCrawlOrdersSubject,
			DefaultCrawlOrdersSubject,
		),
		CrawlOrdersDurable: envconfig.String(
			getenv,
			EnvCrawlOrdersDurable,
			DefaultCrawlOrdersDurable,
		),
		PendingVisitDurable: envconfig.String(
			getenv,
			EnvPendingVisitDurable,
			DefaultPendingVisitDurable,
		),
		ProxyURL:         proxyURL,
		ProxyDialMode:    proxyDialMode,
		FetchConcurrency: limits.fetchConcurrency,
		MaxBodyBytes:     limits.maxBodyBytes,
		FetchDeadline:    limits.fetchDeadline,
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		UserAgent:        envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
		RecrawlGrace:     recrawlGrace,
	}, nil
}

func proxyDialModeFromEnv(getenv func(string) string) (http.ProxyDialMode, error) {
	mode, err := http.ProxyDialModeNamed(
		envconfig.String(getenv, EnvProxyDialMode, DefaultProxyDialMode),
	)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", EnvProxyDialMode, err)
	}
	return mode, nil
}
