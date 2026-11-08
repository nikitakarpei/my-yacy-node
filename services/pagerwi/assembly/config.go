package pagerwiassembly

import (
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPageOfferNATSURL           = "SCRAPE_PAGE_OFFER_NATS_URL"
	EnvPageOfferDurable           = "SCRAPE_PAGE_OFFER_DURABLE"
	EnvPageOfferIntakeConcurrency = "SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY"
	EnvPageRWINATSURL             = "PAGE_RWI_NATS_URL"
	EnvPageRWIMaxBytes            = "PAGE_RWI_MAX_BYTES"
	EnvPageRWIPartitionExponent   = "PAGE_RWI_PARTITION_EXPONENT"
	EnvProvisionStreams           = "PAGERWI_PROVISION_STREAMS"
	EnvOpsAddr                    = "PAGERWI_OPS_ADDR"

	DefaultPageOfferDurable           = "pagerwi"
	DefaultPageOfferIntakeConcurrency = 4
	DefaultPageRWIMaxBytes            = "1GB"
	DefaultPageRWIPartitionExponent   = 4
	DefaultProvisionStreams           = true
	DefaultOpsAddr                    = ":9090"
)

type ServiceConfig struct {
	PageOfferNATSURL           string
	PageOfferDurable           string
	PageOfferIntakeConcurrency int
	PageRWINATSURL             string
	PageRWIMaxBytes            int64
	PageRWIPartitionExponent   uint
	ProvisionsStreams          bool
	OpsAddr                    string
}

func LoadServiceConfig(getenv func(string) string) (ServiceConfig, error) {
	pageOfferNATSURL, err := envconfig.Required(getenv, EnvPageOfferNATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageOfferIntakeConcurrency, err := envconfig.PositiveInt(
		getenv, EnvPageOfferIntakeConcurrency, DefaultPageOfferIntakeConcurrency,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageRWINATSURL, err := envconfig.Required(getenv, EnvPageRWINATSURL)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageRWIMaxBytes, err := envconfig.ByteSize(getenv, EnvPageRWIMaxBytes, DefaultPageRWIMaxBytes)
	if err != nil {
		return ServiceConfig{}, err
	}
	pageRWIPartitionExponent, err := envconfig.PositiveInt(
		getenv, EnvPageRWIPartitionExponent, DefaultPageRWIPartitionExponent,
	)
	if err != nil {
		return ServiceConfig{}, err
	}
	provisionsStreams, err := envconfig.Bool(getenv, EnvProvisionStreams, DefaultProvisionStreams)
	if err != nil {
		return ServiceConfig{}, err
	}
	return ServiceConfig{
		PageOfferNATSURL: pageOfferNATSURL,
		PageOfferDurable: envconfig.String(
			getenv, EnvPageOfferDurable, DefaultPageOfferDurable,
		),
		PageOfferIntakeConcurrency: pageOfferIntakeConcurrency,
		PageRWINATSURL:             pageRWINATSURL,
		PageRWIMaxBytes:            pageRWIMaxBytes,
		PageRWIPartitionExponent:   uint(pageRWIPartitionExponent),
		ProvisionsStreams:          provisionsStreams,
		OpsAddr:                    envconfig.String(getenv, EnvOpsAddr, DefaultOpsAddr),
	}, nil
}
