package main

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	EnvCrawlNATSURL       = "CRAWL_NATS_URL"
	EnvCrawlOrdersSubject = "CRAWL_ORDERS_SUBJECT"

	EnvListenAddr   = "VISITCRAWL_LISTEN_ADDR"
	EnvLinkSecret   = "VISITCRAWL_LINK_SECRET"
	EnvOpsAddr      = "VISITCRAWL_OPS_ADDR"
	EnvOrderTimeout = "VISITCRAWL_ORDER_TIMEOUT"
	EnvMaxInFlight  = "VISITCRAWL_MAX_IN_FLIGHT"
	EnvMaxBodyBytes = "VISITCRAWL_MAX_BODY_BYTES"

	EnvCrawlScope                  = "VISITCRAWL_SCOPE"
	EnvCrawlProfileName            = "VISITCRAWL_PROFILE_NAME"
	EnvCrawlMaxDepth               = "VISITCRAWL_MAX_DEPTH"
	EnvCrawlURLMustMatch           = "VISITCRAWL_URL_MUST_MATCH"
	EnvCrawlURLMustNotMatch        = "VISITCRAWL_URL_MUST_NOT_MATCH"
	EnvCrawlMaxPagesPerHost        = "VISITCRAWL_MAX_PAGES_PER_HOST"
	EnvCrawlAllowQueryURLs         = "VISITCRAWL_ALLOW_QUERY_URLS"
	EnvCrawlIgnoresIndexingRefusal = "VISITCRAWL_IGNORES_INDEXING_REFUSAL"

	DefaultCrawlOrdersSubject          = "yacy.crawl.orders"
	DefaultListenAddr                  = ":8091"
	DefaultOpsAddr                     = ":9091"
	DefaultOrderTimeout                = 5 * time.Second
	DefaultMaxInFlight                 = 256
	DefaultMaxBodyBytes                = 4 << 10
	DefaultCrawlScope                  = "domain"
	DefaultCrawlMaxDepth               = 1
	DefaultCrawlMaxPagesPerHost        = 100
	DefaultCrawlIgnoresIndexingRefusal = true
)

type ServiceConfig struct {
	CrawlNATSURL       string
	CrawlOrdersSubject string
	LinkSecret         string
	ListenAddr         string
	OpsAddr            string
	OrderTimeout       time.Duration
	MaxInFlight        int
	MaxBodyBytes       int64
	CrawlProfile       yacycrawlcontract.CrawlProfile
}

var crawlScopeByName = map[string]yacycrawlcontract.CrawlScope{
	"domain":  yacycrawlcontract.ScopeDomain,
	"wide":    yacycrawlcontract.ScopeWide,
	"subpath": yacycrawlcontract.ScopeSubpath,
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	linkSecret, err := envconfig.Required(getenv, EnvLinkSecret)
	if err != nil {
		return ServiceConfig{}, err
	}

	orderTimeout, err := envconfig.Duration(getenv, EnvOrderTimeout, DefaultOrderTimeout)
	if err != nil {
		return ServiceConfig{}, err
	}
	maxInFlight, err := envconfig.PositiveInt(getenv, EnvMaxInFlight, DefaultMaxInFlight)
	if err != nil {
		return ServiceConfig{}, err
	}
	maxBodyBytes, err := envconfig.PositiveInt64(getenv, EnvMaxBodyBytes, DefaultMaxBodyBytes)
	if err != nil {
		return ServiceConfig{}, err
	}
	profile, err := crawlProfile(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		CrawlNATSURL: crawlNATSURL,
		CrawlOrdersSubject: envconfig.String(
			getenv,
			EnvCrawlOrdersSubject,
			DefaultCrawlOrdersSubject,
		),
		LinkSecret:   linkSecret,
		ListenAddr:   envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:      envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		OrderTimeout: orderTimeout,
		MaxInFlight:  maxInFlight,
		MaxBodyBytes: maxBodyBytes,
		CrawlProfile: profile,
	}, nil
}

func crawlProfile(getenv func(string) string) (yacycrawlcontract.CrawlProfile, error) {
	scopeName := envconfig.String(getenv, EnvCrawlScope, DefaultCrawlScope)
	scope, ok := crawlScopeByName[scopeName]
	if !ok {
		return yacycrawlcontract.CrawlProfile{}, fmt.Errorf(
			"%s: unknown crawl scope %q", EnvCrawlScope, scopeName,
		)
	}

	maxDepth, err := envconfig.NonNegativeInt(getenv, EnvCrawlMaxDepth, DefaultCrawlMaxDepth)
	if err != nil {
		return yacycrawlcontract.CrawlProfile{}, err
	}
	maxPagesPerHost, err := envconfig.Int(
		getenv,
		EnvCrawlMaxPagesPerHost,
		DefaultCrawlMaxPagesPerHost,
	)
	if err != nil {
		return yacycrawlcontract.CrawlProfile{}, err
	}
	if maxPagesPerHost != yacycrawlcontract.UnlimitedPagesPerHost && maxPagesPerHost <= 0 {
		return yacycrawlcontract.CrawlProfile{}, fmt.Errorf(
			"%s: must be positive or %d for unlimited",
			EnvCrawlMaxPagesPerHost, yacycrawlcontract.UnlimitedPagesPerHost,
		)
	}
	allowQueryURLs, err := envconfig.Bool(getenv, EnvCrawlAllowQueryURLs, false)
	if err != nil {
		return yacycrawlcontract.CrawlProfile{}, err
	}
	ignoresIndexingRefusal, err := envconfig.Bool(
		getenv,
		EnvCrawlIgnoresIndexingRefusal,
		DefaultCrawlIgnoresIndexingRefusal,
	)
	if err != nil {
		return yacycrawlcontract.CrawlProfile{}, err
	}

	return yacycrawlcontract.CrawlProfile{
		Name:                   envconfig.String(getenv, EnvCrawlProfileName, ""),
		Scope:                  scope,
		URLMustMatch:           matchOrAll(envconfig.String(getenv, EnvCrawlURLMustMatch, "")),
		URLMustNotMatch:        envconfig.String(getenv, EnvCrawlURLMustNotMatch, ""),
		MaxDepth:               maxDepth,
		AllowQueryURLs:         allowQueryURLs,
		MaxPagesPerHost:        maxPagesPerHost,
		IgnoresIndexingRefusal: ignoresIndexingRefusal,
	}, nil
}

func matchOrAll(pattern string) string {
	if pattern == "" {
		return yacycrawlcontract.MatchAll
	}
	return pattern
}
