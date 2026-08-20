package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	EnvCrawlNATSURL           = "CRAWL_NATS_URL"
	EnvPageMarkdownNATSURL    = "PAGE_MARKDOWN_NATS_URL"
	EnvNATSCrawledPageSubject = "NATS_CRAWLED_PAGE_SUBJECT"
	EnvNATSCrawledPageDurable = "NATS_CRAWLED_PAGE_DURABLE"
	EnvConcurrency            = "CORPUSMARKDOWN_CONCURRENCY"
	EnvOpsAddr                = "CORPUSMARKDOWN_OPS_ADDR"

	DefaultOpsAddr            = ":9090"
	DefaultCrawledPageDurable = "corpusmarkdown"
	DefaultConcurrency        = 4
)

var DefaultCrawledPageSubject = yacycrawlcontract.CrawledPageSubject(
	yacycrawlcontract.PageRepresentationKindMarkdown,
)

type ServiceConfig struct {
	CrawlNATSURL        string
	PageMarkdownNATSURL string
	CrawledPageSubject  string
	CrawledPageDurable  string
	Concurrency         int
	OpsAddr             string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	crawlNATSURL, err := envconfig.Required(getenv, EnvCrawlNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	pageMarkdownNATSURL, err := envconfig.Required(getenv, EnvPageMarkdownNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	concurrency, err := envconfig.PositiveInt(getenv, EnvConcurrency, DefaultConcurrency)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		CrawlNATSURL:        crawlNATSURL,
		PageMarkdownNATSURL: pageMarkdownNATSURL,
		CrawledPageSubject: envconfig.String(
			getenv,
			EnvNATSCrawledPageSubject,
			DefaultCrawledPageSubject,
		),
		CrawledPageDurable: envconfig.String(
			getenv,
			EnvNATSCrawledPageDurable,
			DefaultCrawledPageDurable,
		),
		Concurrency: concurrency,
		OpsAddr:     envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
