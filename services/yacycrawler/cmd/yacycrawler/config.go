package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
)

const (
	EnvCrawlNATSURL  = "CRAWL_NATS_URL"
	EnvOrdersSubject = "NATS_ORDERS_SUBJECT"
	EnvOrdersDurable = "NATS_ORDERS_DURABLE"

	EnvProxyURL         = "YACYCRAWLER_PROXY_URL"
	EnvProxyDialMode    = "YACYCRAWLER_PROXY_DIAL_MODE"
	EnvFetchConcurrency = "YACYCRAWLER_FETCH_CONCURRENCY"

	EnvRunPageBudget = "YACYCRAWLER_RUN_PAGE_BUDGET"
	EnvFrontierCap   = "YACYCRAWLER_FRONTIER_CAP"
	EnvMaxBodyBytes  = "YACYCRAWLER_MAX_BODY_BYTES"
	EnvFetchDeadline = "YACYCRAWLER_FETCH_DEADLINE"
	EnvContentTypes  = "YACYCRAWLER_CONTENT_TYPES"
	EnvOpsAddr       = "YACYCRAWLER_OPS_ADDR"
	EnvListenAddr    = "YACYCRAWLER_LISTEN_ADDR"
	EnvUserAgent     = "YACYCRAWLER_USER_AGENT"
	EnvRecrawlGrace  = "YACYCRAWLER_RECRAWL_GRACE"

	DefaultOrdersSubject    = "yacy.crawl.orders"
	DefaultOrdersDurable    = "yacycrawler"
	DefaultMaxMsgs          = 1024
	DefaultFetchConcurrency = 4
	DefaultRunPageBudget    = 1000
	DefaultFrontierCap      = 10000
	DefaultMaxBodyBytes     = 2 << 20
	DefaultFetchDeadline    = 30 * time.Second
	DefaultOpsAddr          = ":9090"
	DefaultListenAddr       = ":8095"
	DefaultUserAgent        = "yacycrawler (+https://yacy.net)"
	DefaultProxyDialMode    = "tunnel"

	DefaultRedirectResolutionMaxBytes = 256 << 20

	DefaultRecrawlGrace       = time.Hour
	DefaultPageVisitRetention = 30 * 24 * time.Hour
	DefaultPageVisitMaxBytes  = 256 << 20

	DefaultDisposedPagesRetention = 24 * time.Hour
	DefaultDisposedPagesMaxBytes  = 256 << 20
)

type ServiceConfig struct {
	CrawlNATSURL     string
	OrdersSubject    string
	OrdersDurable    string
	ProxyURL         *url.URL
	ProxyDialMode    http.ProxyDialMode
	FetchConcurrency int
	RunPageBudget    int
	FrontierCap      int
	MaxBodyBytes     int64
	FetchDeadline    time.Duration
	ContentTypes     []string
	OpsAddr          string
	ListenAddr       string
	UserAgent        string
	RecrawlGrace     time.Duration
}

func (ServiceConfig) PageVisitBucketSpec() dueaftergrace.BucketSpec {
	return dueaftergrace.BucketSpec{
		MaxBytes:  DefaultPageVisitMaxBytes,
		Retention: DefaultPageVisitRetention,
	}
}

type serviceLimits struct {
	fetchConcurrency int
	runPageBudget    int
	frontierCap      int
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
	runPageBudget, err := envconfig.PositiveInt(getenv, EnvRunPageBudget, DefaultRunPageBudget)
	if err != nil {
		return serviceLimits{}, err
	}
	frontierCap, err := envconfig.PositiveInt(getenv, EnvFrontierCap, DefaultFrontierCap)
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
		runPageBudget:    runPageBudget,
		frontierCap:      frontierCap,
		maxBodyBytes:     maxBodyBytes,
		fetchDeadline:    fetchDeadline,
	}, nil
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
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
		CrawlNATSURL:     crawlNATSURL,
		OrdersSubject:    envconfig.String(getenv, EnvOrdersSubject, DefaultOrdersSubject),
		OrdersDurable:    envconfig.String(getenv, EnvOrdersDurable, DefaultOrdersDurable),
		ProxyURL:         proxyURL,
		ProxyDialMode:    proxyDialMode,
		FetchConcurrency: limits.fetchConcurrency,
		RunPageBudget:    limits.runPageBudget,
		FrontierCap:      limits.frontierCap,
		MaxBodyBytes:     limits.maxBodyBytes,
		FetchDeadline:    limits.fetchDeadline,
		ContentTypes:     mediaTypes(getenv, EnvContentTypes),
		OpsAddr:          envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		ListenAddr:       envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
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

func mediaTypes(getenv func(string) string, key string) []string {
	values := envconfig.List(getenv, key)
	for i, value := range values {
		values[i] = strings.ToLower(value)
	}
	return values
}
