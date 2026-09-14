package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPageOfferNATSURL           = "SCRAPE_PAGE_OFFER_NATS_URL"
	EnvPageMarkdownNATSURL        = "PAGE_MARKDOWN_NATS_URL"
	EnvPageOfferDurable           = "SCRAPE_PAGE_OFFER_DURABLE"
	EnvPageOfferIntakeConcurrency = "SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY"
	EnvListenAddr                 = "CORPUSMARKDOWN_LISTEN_ADDR"
	EnvOpsAddr                    = "CORPUSMARKDOWN_OPS_ADDR"

	DefaultListenAddr                 = ":8094"
	DefaultOpsAddr                    = ":9090"
	DefaultPageOfferDurable           = "corpusmarkdown"
	DefaultPageOfferIntakeConcurrency = 4
)

type ServiceConfig struct {
	PageOfferNATSURL           string
	PageMarkdownNATSURL        string
	PageOfferDurable           string
	PageOfferIntakeConcurrency int
	ListenAddr                 string
	OpsAddr                    string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	pageOfferNATSURL, err := envconfig.Required(getenv, EnvPageOfferNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageMarkdownNATSURL, err := envconfig.Required(getenv, EnvPageMarkdownNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageOfferIntakeConcurrency, err := envconfig.PositiveInt(
		getenv,
		EnvPageOfferIntakeConcurrency,
		DefaultPageOfferIntakeConcurrency,
	)
	if err != nil {
		return ServiceConfig{}, err
	}

	return ServiceConfig{
		PageOfferNATSURL:    pageOfferNATSURL,
		PageMarkdownNATSURL: pageMarkdownNATSURL,
		PageOfferDurable: envconfig.String(
			getenv,
			EnvPageOfferDurable,
			DefaultPageOfferDurable,
		),
		PageOfferIntakeConcurrency: pageOfferIntakeConcurrency,
		ListenAddr:                 envconfig.String(getenv, EnvListenAddr, DefaultListenAddr),
		OpsAddr:                    envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
