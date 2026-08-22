package main

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvCrawlNATSURL  = "CRAWL_NATS_URL"
	EnvOrdersSubject = "NATS_ORDERS_SUBJECT"

	EnvListenAddr           = "FIRECRAWLSHIM_LISTEN_ADDR"
	EnvCrawlOutcomesTarget  = "FIRECRAWLSHIM_CRAWL_OUTCOMES_TARGET"
	EnvMarkdownCorpusTarget = "FIRECRAWLSHIM_MARKDOWN_CORPUS_TARGET"
	EnvRecallLimit          = "FIRECRAWLSHIM_RECALL_LIMIT"
	EnvPollInterval         = "FIRECRAWLSHIM_POLL_INTERVAL"
	EnvMaxInFlight          = "FIRECRAWLSHIM_MAX_IN_FLIGHT"

	DefaultOrdersSubject = "yacy.crawl.orders"
	DefaultListenAddr    = ":8093"
	DefaultRecallLimit   = 30 * time.Second
	DefaultPollInterval  = 500 * time.Millisecond
	DefaultMaxInFlight   = 256
)

type ServiceConfig struct {
	CrawlNATSURL         string
	OrdersSubject        string
	ListenAddr           string
	CrawlOutcomesTarget  string
	MarkdownCorpusTarget string
	RecallLimit          time.Duration
	PollInterval         time.Duration
	MaxInFlight          int
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	crawlOutcomesTarget, err := envconfig.Required(getenv, EnvCrawlOutcomesTarget)
	if err != nil {
		return ServiceConfig{}, err
	}
	markdownCorpusTarget, err := envconfig.Required(getenv, EnvMarkdownCorpusTarget)
	if err != nil {
		return ServiceConfig{}, err
	}
	recallLimit, err := envconfig.Duration(getenv, EnvRecallLimit, DefaultRecallLimit)
	if err != nil {
		return ServiceConfig{}, err
	}
	pollInterval, err := envconfig.Duration(getenv, EnvPollInterval, DefaultPollInterval)
	if err != nil {
		return ServiceConfig{}, err
	}
	maxInFlight, err := envconfig.PositiveInt(getenv, EnvMaxInFlight, DefaultMaxInFlight)
	if err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		CrawlNATSURL:         crawlNATSURL,
		OrdersSubject:        envconfig.String(getenv, EnvOrdersSubject, DefaultOrdersSubject),
		ListenAddr:           envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		CrawlOutcomesTarget:  crawlOutcomesTarget,
		MarkdownCorpusTarget: markdownCorpusTarget,
		RecallLimit:          recallLimit,
		PollInterval:         pollInterval,
		MaxInFlight:          maxInFlight,
	}, nil
}
