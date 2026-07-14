package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/httpfetch"
)

const (
	EnvNATSURL             = "NATS_URL"
	EnvOrdersSubject       = "NATS_ORDERS_SUBJECT"
	EnvOrdersDurable       = "NATS_ORDERS_DURABLE"
	EnvPageRWISubject      = "NATS_PAGE_RWI_SUBJECT"
	EnvPageRWIMaxMsgs      = "NATS_PAGE_RWI_MAX_MSGS"
	EnvPageTextSubject     = "NATS_PAGE_TEXT_SUBJECT"
	EnvPageTextMaxMsgs     = "NATS_PAGE_TEXT_MAX_MSGS"
	EnvPageMarkdownSubject = "NATS_PAGE_MARKDOWN_SUBJECT"
	EnvPageMarkdownMaxMsgs = "NATS_PAGE_MARKDOWN_MAX_MSGS"

	EnvProxyURL         = "YACYCRAWLER_PROXY_URL"
	EnvProxyDialMode    = "YACYCRAWLER_PROXY_DIAL_MODE"
	EnvFetchConcurrency = "YACYCRAWLER_FETCH_CONCURRENCY"

	EnvRWIOutputEnabled      = "YACYCRAWLER_RWI_OUTPUT_ENABLED"
	EnvTextOutputEnabled     = "YACYCRAWLER_TEXT_OUTPUT_ENABLED"
	EnvMarkdownOutputEnabled = "YACYCRAWLER_MARKDOWN_OUTPUT_ENABLED"

	EnvRunPageBudget = "YACYCRAWLER_RUN_PAGE_BUDGET"
	EnvFrontierCap   = "YACYCRAWLER_FRONTIER_CAP"
	EnvMaxBodyBytes  = "YACYCRAWLER_MAX_BODY_BYTES"
	EnvFetchDeadline = "YACYCRAWLER_FETCH_DEADLINE"
	EnvContentTypes  = "YACYCRAWLER_CONTENT_TYPES"
	EnvOpsAddr       = "YACYCRAWLER_OPS_ADDR"
	EnvUserAgent     = "YACYCRAWLER_USER_AGENT"

	DefaultOrdersSubject       = "yacy.crawl.orders"
	DefaultOrdersDurable       = "yacycrawler"
	DefaultPageRWISubject      = "yacy.crawl.page.rwi"
	DefaultPageTextSubject     = "yacy.crawl.page.text"
	DefaultPageMarkdownSubject = "yacy.crawl.page.markdown"
	DefaultMaxMsgs             = 1024
	DefaultFetchConcurrency    = 4
	DefaultRunPageBudget       = 1000
	DefaultFrontierCap         = 10000
	DefaultMaxBodyBytes        = 2 << 20
	DefaultFetchDeadline       = 30 * time.Second
	DefaultOpsAddr             = ":9090"
	DefaultUserAgent           = "yacycrawler (+https://yacy.net)"
	DefaultProxyDialMode       = "tunnel"
)

var proxyDialModeByName = map[string]httpfetch.ProxyDialMode{
	"tunnel":       httpfetch.ProxyDialTunnel,
	"absolute-url": httpfetch.ProxyDialAbsoluteURL,
}

type ServiceConfig struct {
	NATSURL               string
	OrdersSubject         string
	OrdersDurable         string
	PageRWISubject        string
	PageRWIMaxMsgs        int64
	PageTextSubject       string
	PageTextMaxMsgs       int64
	PageMarkdownSubject   string
	PageMarkdownMaxMsgs   int64
	ProxyURL              *url.URL
	ProxyDialMode         httpfetch.ProxyDialMode
	FetchConcurrency      int
	RWIOutputEnabled      bool
	TextOutputEnabled     bool
	MarkdownOutputEnabled bool
	RunPageBudget         int
	FrontierCap           int
	MaxBodyBytes          int64
	FetchDeadline         time.Duration
	ContentTypes          []string
	OpsAddr               string
	UserAgent             string
}

func (c ServiceConfig) OrdersStreamSpec() yacycrawlcontract.OrdersStreamSpec {
	return yacycrawlcontract.OrdersStreamSpec{Subject: c.OrdersSubject}
}

func (c ServiceConfig) PageRWIStreamSpec() yacycrawlcontract.CrawledPageStreamSpec {
	return yacycrawlcontract.CrawledPageStreamSpec{
		Subject: c.PageRWISubject,
		MaxMsgs: c.PageRWIMaxMsgs,
	}
}

func (c ServiceConfig) PageTextStreamSpec() yacycrawlcontract.CrawledPageStreamSpec {
	return yacycrawlcontract.CrawledPageStreamSpec{
		Subject: c.PageTextSubject,
		MaxMsgs: c.PageTextMaxMsgs,
	}
}

func (c ServiceConfig) PageMarkdownStreamSpec() yacycrawlcontract.CrawledPageStreamSpec {
	return yacycrawlcontract.CrawledPageStreamSpec{
		Subject: c.PageMarkdownSubject,
		MaxMsgs: c.PageMarkdownMaxMsgs,
	}
}

type serviceLimits struct {
	pageRWIMaxMsgs      int64
	pageTextMaxMsgs     int64
	pageMarkdownMaxMsgs int64
	fetchConcurrency    int
	runPageBudget       int
	frontierCap         int
	maxBodyBytes        int64
	fetchDeadline       time.Duration
	rwiEnabled          bool
	textEnabled         bool
	markdownEnabled     bool
}

func loadServiceLimits(getenv func(string) string) (serviceLimits, error) {
	pageRWIMaxMsgs, err := envconfig.PositiveInt64(getenv, EnvPageRWIMaxMsgs, DefaultMaxMsgs)
	if err != nil {
		return serviceLimits{}, err
	}
	pageTextMaxMsgs, err := envconfig.PositiveInt64(getenv, EnvPageTextMaxMsgs, DefaultMaxMsgs)
	if err != nil {
		return serviceLimits{}, err
	}
	pageMarkdownMaxMsgs, err := envconfig.PositiveInt64(
		getenv,
		EnvPageMarkdownMaxMsgs,
		DefaultMaxMsgs,
	)
	if err != nil {
		return serviceLimits{}, err
	}
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
	rwiEnabled, err := envconfig.Bool(getenv, EnvRWIOutputEnabled, true)
	if err != nil {
		return serviceLimits{}, err
	}
	textEnabled, err := envconfig.Bool(getenv, EnvTextOutputEnabled, false)
	if err != nil {
		return serviceLimits{}, err
	}
	markdownEnabled, err := envconfig.Bool(getenv, EnvMarkdownOutputEnabled, false)
	if err != nil {
		return serviceLimits{}, err
	}
	if !rwiEnabled && !textEnabled && !markdownEnabled {
		return serviceLimits{}, fmt.Errorf(
			"at least one of %s, %s or %s must be enabled",
			EnvRWIOutputEnabled, EnvTextOutputEnabled, EnvMarkdownOutputEnabled,
		)
	}

	return serviceLimits{
		pageRWIMaxMsgs:      pageRWIMaxMsgs,
		pageTextMaxMsgs:     pageTextMaxMsgs,
		pageMarkdownMaxMsgs: pageMarkdownMaxMsgs,
		fetchConcurrency:    fetchConcurrency,
		runPageBudget:       runPageBudget,
		frontierCap:         frontierCap,
		maxBodyBytes:        maxBodyBytes,
		fetchDeadline:       fetchDeadline,
		rwiEnabled:          rwiEnabled,
		textEnabled:         textEnabled,
		markdownEnabled:     markdownEnabled,
	}, nil
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	natsURL := strings.TrimSpace(getenv(EnvNATSURL))
	if natsURL == "" {
		return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvNATSURL)
	}
	proxyURL, err := requiredURL(getenv, EnvProxyURL)
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

	return ServiceConfig{
		NATSURL:         natsURL,
		OrdersSubject:   envconfig.String(getenv, EnvOrdersSubject, DefaultOrdersSubject),
		OrdersDurable:   envconfig.String(getenv, EnvOrdersDurable, DefaultOrdersDurable),
		PageRWISubject:  envconfig.String(getenv, EnvPageRWISubject, DefaultPageRWISubject),
		PageRWIMaxMsgs:  limits.pageRWIMaxMsgs,
		PageTextSubject: envconfig.String(getenv, EnvPageTextSubject, DefaultPageTextSubject),
		PageTextMaxMsgs: limits.pageTextMaxMsgs,
		PageMarkdownSubject: envconfig.String(
			getenv,
			EnvPageMarkdownSubject,
			DefaultPageMarkdownSubject,
		),
		PageMarkdownMaxMsgs:   limits.pageMarkdownMaxMsgs,
		ProxyURL:              proxyURL,
		ProxyDialMode:         proxyDialMode,
		FetchConcurrency:      limits.fetchConcurrency,
		RWIOutputEnabled:      limits.rwiEnabled,
		TextOutputEnabled:     limits.textEnabled,
		MarkdownOutputEnabled: limits.markdownEnabled,
		RunPageBudget:         limits.runPageBudget,
		FrontierCap:           limits.frontierCap,
		MaxBodyBytes:          limits.maxBodyBytes,
		FetchDeadline:         limits.fetchDeadline,
		ContentTypes:          mediaTypes(getenv, EnvContentTypes),
		OpsAddr:               envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		UserAgent:             envconfig.String(getenv, EnvUserAgent, DefaultUserAgent),
	}, nil
}

func requiredURL(getenv func(string) string, key string) (*url.URL, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return nil, fmt.Errorf("%s: must be set", key)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s: scheme must be http or https", key)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("%s: must include a host", key)
	}
	return parsed, nil
}

func proxyDialModeFromEnv(getenv func(string) string) (httpfetch.ProxyDialMode, error) {
	name := envconfig.String(getenv, EnvProxyDialMode, DefaultProxyDialMode)
	mode, ok := proxyDialModeByName[name]
	if !ok {
		return 0, fmt.Errorf("%s: unknown proxy dial mode %q", EnvProxyDialMode, name)
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
