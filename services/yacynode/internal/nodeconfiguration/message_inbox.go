package nodeconfiguration

import (
	"fmt"
	"net/url"
	"strings"
)

const EnvMessageWebhookURL = "YACY_MESSAGE_WEBHOOK_URL"

type MessageInboxConfig struct {
	WebhookURL *url.URL
}

func (c MessageInboxConfig) Enabled() bool {
	return c.WebhookURL != nil
}

func loadMessageInboxConfig(getenv func(string) string) (MessageInboxConfig, error) {
	raw := strings.TrimSpace(getenv(EnvMessageWebhookURL))
	if raw == "" {
		return MessageInboxConfig{}, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return MessageInboxConfig{}, fmt.Errorf("%s: not a URL", EnvMessageWebhookURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return MessageInboxConfig{}, fmt.Errorf(
			"%s: scheme must be http or https",
			EnvMessageWebhookURL,
		)
	}
	if parsed.Hostname() == "" {
		return MessageInboxConfig{}, fmt.Errorf("%s: must include a host", EnvMessageWebhookURL)
	}

	return MessageInboxConfig{WebhookURL: parsed}, nil
}
