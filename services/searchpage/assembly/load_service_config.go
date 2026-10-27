package searchpageassembly

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pageshell"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvListenAddr             = "SEARCHPAGE_LISTEN_ADDR"
	EnvCorpusTextURL          = "SEARCHPAGE_CORPUSTEXT_URL"
	EnvYaCySearchURL          = "SEARCHPAGE_YACYSEARCH_URL"
	EnvYaCySearchCallDeadline = "SEARCHPAGE_YACYSEARCH_CALL_DEADLINE"
	EnvVisitCrawlURL          = "SEARCHPAGE_VISITCRAWL_URL"
	EnvVisitLinkSecret        = "VISITCRAWL_LINK_SECRET"
	EnvVisitLinkLifetime      = "SEARCHPAGE_VISIT_LINK_LIFETIME"
	EnvTabBar                 = "SEARCHPAGE_TAB_BAR"
	EnvOpsAddr                = "SEARCHPAGE_OPS_ADDR"
)

const (
	DefaultListenAddr             = ":8097"
	DefaultYaCySearchCallDeadline = 12 * time.Second
	DefaultVisitLinkLifetime      = 24 * time.Hour
	DefaultOpsAddr                = ":9090"
)

var (
	errNoSource         = errors.New("set at least one of them")
	errHalfOfVisitCrawl = errors.New("set both or neither")
)

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	corpusTextURL, err := optionalHTTPURL(getenv, EnvCorpusTextURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	yacySearchURL, err := optionalHTTPURL(getenv, EnvYaCySearchURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	if corpusTextURL == nil && yacySearchURL == nil {
		return ServiceConfig{}, fmt.Errorf(
			"%s, %s: %w", EnvCorpusTextURL, EnvYaCySearchURL, errNoSource,
		)
	}
	yacySearchCallDeadline, err := envconfig.Duration(
		getenv, EnvYaCySearchCallDeadline, DefaultYaCySearchCallDeadline,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	visitCrawl, err := visitCrawlConfigFrom(getenv)
	if err != nil {
		return ServiceConfig{}, err
	}
	tabBar, err := pageshell.TabBarFrom(getenv(EnvTabBar))
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("%s: %w", EnvTabBar, err)
	}

	return ServiceConfig{
		CorpusTextURL:          corpusTextURL,
		YaCySearchURL:          yacySearchURL,
		VisitCrawl:             visitCrawl,
		TabBar:                 tabBar,
		ListenAddr:             envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:                envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
		YaCySearchCallDeadline: yacySearchCallDeadline,
	}, nil
}

func optionalHTTPURL(getenv func(string) string, key string) (*url.URL, error) {
	if strings.TrimSpace(getenv(key)) == "" {
		return nil, nil
	}
	return envconfig.RequiredHTTPURL(getenv, key)
}

func visitCrawlConfigFrom(getenv func(string) string) (*VisitCrawlConfig, error) {
	visitCrawlURL, err := optionalHTTPURL(getenv, EnvVisitCrawlURL)
	if err != nil {
		return nil, err
	}
	linkSecret := strings.TrimSpace(getenv(EnvVisitLinkSecret))
	if visitCrawlURL == nil && linkSecret == "" {
		return nil, nil
	}
	if visitCrawlURL == nil || linkSecret == "" {
		return nil, fmt.Errorf(
			"%s, %s: %w", EnvVisitCrawlURL, EnvVisitLinkSecret, errHalfOfVisitCrawl,
		)
	}
	linkLifetime, err := envconfig.Duration(getenv, EnvVisitLinkLifetime, DefaultVisitLinkLifetime)
	if err != nil {
		return nil, err
	}
	return &VisitCrawlConfig{
		URL:          visitCrawlURL,
		LinkSecret:   linkSecret,
		LinkLifetime: linkLifetime,
	}, nil
}
