package nodeconfiguration

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/envconfig"
)

const (
	EnvPageRWINATSURL           = "PAGE_RWI_NATS_URL"
	EnvPageRWIDurable           = "PAGE_RWI_DURABLE"
	EnvPageRWIIntakeConcurrency = "PAGE_RWI_INTAKE_CONCURRENCY"

	DefaultPageRWIDurable           = "yacy-node"
	DefaultPageRWIIntakeConcurrency = 4
)

type PageRWIIntakeConfig struct {
	PageRWINATSURL           string
	PageRWIDurable           string
	PageRWIIntakeConcurrency int
}

func (c PageRWIIntakeConfig) Enabled() bool {
	return c.PageRWINATSURL != ""
}

func loadPageRWIIntakeConfig(getenv func(string) string) (PageRWIIntakeConfig, error) {
	pageRWINATSURL := strings.TrimSpace(getenv(EnvPageRWINATSURL))
	if pageRWINATSURL == "" {
		return PageRWIIntakeConfig{}, nil
	}
	pageRWIIntakeConcurrency, err := envconfig.PositiveInt(
		getenv, EnvPageRWIIntakeConcurrency, DefaultPageRWIIntakeConcurrency,
	)
	if err != nil {
		return PageRWIIntakeConfig{}, err
	}

	return PageRWIIntakeConfig{
		PageRWINATSURL: pageRWINATSURL,
		PageRWIDurable: envconfig.String(
			getenv, EnvPageRWIDurable, DefaultPageRWIDurable,
		),
		PageRWIIntakeConcurrency: pageRWIIntakeConcurrency,
	}, nil
}
