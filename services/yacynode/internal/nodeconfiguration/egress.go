package nodeconfiguration

import (
	"fmt"
	"net/url"
	"strings"
)

const EnvEgressProxyURL = "EGRESS_PROXY_URL"

type EgressConfig struct {
	ProxyURL *url.URL
}

func loadEgressConfig(getenv func(string) string) (EgressConfig, error) {
	raw := strings.TrimSpace(getenv(EnvEgressProxyURL))
	if raw == "" {
		return EgressConfig{}, fmt.Errorf("%s: must be set", EnvEgressProxyURL)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return EgressConfig{}, fmt.Errorf("%s: %w", EnvEgressProxyURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return EgressConfig{}, fmt.Errorf("%s: scheme must be http or https", EnvEgressProxyURL)
	}
	if parsed.Host == "" {
		return EgressConfig{}, fmt.Errorf("%s: must include a host", EnvEgressProxyURL)
	}

	return EgressConfig{ProxyURL: parsed}, nil
}
