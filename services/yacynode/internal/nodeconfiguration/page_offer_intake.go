package nodeconfiguration

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPageOfferNATSURL           = "SCRAPE_PAGE_OFFER_NATS_URL"
	EnvPageOfferDurable           = "SCRAPE_PAGE_OFFER_DURABLE"
	EnvPageOfferIntakeConcurrency = "SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY"

	DefaultPageOfferDurable           = "yacy-node"
	DefaultPageOfferIntakeConcurrency = 4
)

type PageOfferIntakeConfig struct {
	PageOfferNATSURL           string
	PageOfferDurable           string
	PageOfferIntakeConcurrency int
}

func (c PageOfferIntakeConfig) Enabled() bool {
	return c.PageOfferNATSURL != ""
}

func loadPageOfferIntakeConfig(getenv func(string) string) (PageOfferIntakeConfig, error) {
	pageOfferNATSURL := strings.TrimSpace(getenv(EnvPageOfferNATSURL))
	if pageOfferNATSURL == "" {
		return PageOfferIntakeConfig{}, nil
	}
	pageOfferIntakeConcurrency, err := envconfig.PositiveInt(
		getenv, EnvPageOfferIntakeConcurrency, DefaultPageOfferIntakeConcurrency,
	)
	if err != nil {
		return PageOfferIntakeConfig{}, err
	}

	return PageOfferIntakeConfig{
		PageOfferNATSURL: pageOfferNATSURL,
		PageOfferDurable: envconfig.String(
			getenv, EnvPageOfferDurable, DefaultPageOfferDurable,
		),
		PageOfferIntakeConcurrency: pageOfferIntakeConcurrency,
	}, nil
}
