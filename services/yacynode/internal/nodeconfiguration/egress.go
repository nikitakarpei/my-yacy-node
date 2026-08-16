package nodeconfiguration

import (
	"fmt"
	"net/url"
	"strings"
)

const EnvProxyURL = "YACY_PROXY_URL"

type EgressConfig struct {
	ProxyURL *url.URL
}

func loadEgressConfig(getenv func(string) string) (EgressConfig, error) {
	raw := strings.TrimSpace(getenv(EnvProxyURL))
	if raw == "" {
		return EgressConfig{}, fmt.Errorf("%s: must be set", EnvProxyURL)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return EgressConfig{}, fmt.Errorf("%s: %w", EnvProxyURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return EgressConfig{}, fmt.Errorf("%s: scheme must be http or https", EnvProxyURL)
	}
	if parsed.Host == "" {
		return EgressConfig{}, fmt.Errorf("%s: must include a host", EnvProxyURL)
	}

	return EgressConfig{ProxyURL: parsed}, nil
}
