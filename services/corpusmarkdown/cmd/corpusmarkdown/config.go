package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	EnvNATSURL                = "NATS_URL"
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
	NATSURL            string
	CrawledPageSubject string
	CrawledPageDurable string
	Concurrency        int
	OpsAddr            string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	natsURL, err := envconfig.Required(getenv, EnvNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}

	concurrency, err := envconfig.PositiveInt(getenv, EnvConcurrency, DefaultConcurrency)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		NATSURL: natsURL,
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
