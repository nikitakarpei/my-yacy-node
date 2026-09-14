//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxng"
)

const (
	searxngAlias = "searxng"
	yacyEngine   = "yacy"
	yacyBang     = "ya"
)

func startSearXNG(
	t *testing.T,
	ctx context.Context,
	networkName string,
	searchAlias string,
) string {
	t.Helper()

	return searxng.Start(t, ctx, networkName, searxng.Config{
		Alias:        searxngAlias,
		SettingsYAML: settingsYAMLFor(searchAlias),
	})
}

func settingsYAMLFor(searchAlias string) string {
	return `use_default_settings:
  engines:
    keep_only:
      - ` + yacyEngine + `

server:
  secret_key: "e2e-test-secret-key"

search:
  formats:
    - html
    - json

outgoing:
  request_timeout: 20.0
  max_request_timeout: 30.0

engines:
  - name: ` + yacyEngine + `
    engine: ` + yacyEngine + `
    shortcut: ` + yacyBang + `
    categories: general
    search_type: text
    disabled: false
    enable_http: true
    timeout: 20.0
    base_url: "http://` + searchAlias + `:` + yacydhtsearchPort + `"
`
}
