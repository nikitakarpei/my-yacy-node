package main

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPageOfferNATSURL           = "SCRAPE_PAGE_OFFER_NATS_URL"
	EnvPageOfferDurable           = "SCRAPE_PAGE_OFFER_DURABLE"
	EnvPageOfferIntakeConcurrency = "SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY"
	EnvSearchIndexEngine          = "SEARCH_INDEX_ENGINE"
	EnvElasticsearchURL           = "ELASTICSEARCH_URL"
	EnvElasticsearchIndex         = "ELASTICSEARCH_INDEX"
	EnvManticoreURL               = "MANTICORE_URL"
	EnvManticoreTable             = "MANTICORE_TABLE"
	EnvLanguages                  = "CORPUSTEXT_LANGUAGES"
	EnvOpsAddr                    = "CORPUSTEXT_OPS_ADDR"

	DefaultOpsAddr                    = ":9090"
	DefaultPageOfferDurable           = "corpustext"
	DefaultPageOfferIntakeConcurrency = 4
	DefaultIndexBaseName              = "yacy_text"

	SearchIndexEngineElasticsearch = "elasticsearch"
	SearchIndexEngineManticore     = "manticore"
)

type ServiceConfig struct {
	PageOfferNATSURL           string
	PageOfferDurable           string
	PageOfferIntakeConcurrency int
	SearchIndexEngine          string
	ElasticsearchURL           string
	ElasticsearchIndex         string
	ManticoreURL               string
	ManticoreTable             string
	Languages                  []string
	OpsAddr                    string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	pageOfferNATSURL, err := envconfig.Required(getenv, EnvPageOfferNATSURL)
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

	cfg := ServiceConfig{
		PageOfferNATSURL: pageOfferNATSURL,
		PageOfferDurable: envconfig.String(
			getenv,
			EnvPageOfferDurable,
			DefaultPageOfferDurable,
		),
		PageOfferIntakeConcurrency: pageOfferIntakeConcurrency,
		SearchIndexEngine:          strings.TrimSpace(getenv(EnvSearchIndexEngine)),
		Languages:                  envconfig.List(getenv, EnvLanguages),
		OpsAddr:                    envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}
	if cfg.SearchIndexEngine == "" {
		return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvSearchIndexEngine)
	}

	switch cfg.SearchIndexEngine {
	case SearchIndexEngineElasticsearch:
		cfg.ElasticsearchURL = strings.TrimSpace(getenv(EnvElasticsearchURL))
		if cfg.ElasticsearchURL == "" {
			return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvElasticsearchURL)
		}
		cfg.ElasticsearchIndex = envconfig.String(
			getenv,
			EnvElasticsearchIndex,
			DefaultIndexBaseName,
		)
	case SearchIndexEngineManticore:
		cfg.ManticoreURL = strings.TrimSpace(getenv(EnvManticoreURL))
		if cfg.ManticoreURL == "" {
			return ServiceConfig{}, fmt.Errorf("%s: must be set", EnvManticoreURL)
		}
		cfg.ManticoreTable = envconfig.String(getenv, EnvManticoreTable, DefaultIndexBaseName)
	default:
		return ServiceConfig{}, unknownSearchIndexEngine(cfg.SearchIndexEngine)
	}

	return cfg, nil
}

func unknownSearchIndexEngine(engine string) error {
	return fmt.Errorf("%s: unknown engine %q", EnvSearchIndexEngine, engine)
}
