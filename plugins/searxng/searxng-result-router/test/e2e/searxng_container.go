//go:build e2e

package e2e

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxng"
)

const (
	searxngAlias       = "searxng"
	pluginMountDir     = "/opt/e2e-plugins"
	pluginSourcePath   = "../../result_link_router.py"
	engineMountDir     = "/usr/local/searxng/searx/engines"
	testEngineModule   = "offline_test_engine"
	testEngineName     = "origin"
	testEngineBang     = "ot"
	destinationPageURL = "http://example.invalid/destination-page"
)

func testEngineSource(searchUpstreamURL string) string {
	return `categories = ["general"]
about = {}


def request(query, params):
    params["url"] = "` + searchUpstreamURL + `"
    return params


def response(resp):
    return [
        {
            "title": "Destination page",
            "url": "` + destinationPageURL + `",
            "content": "destination page content",
        }
    ]
`
}

const testSettingsYAML = `use_default_settings:
  engines:
    keep_only:
      - ` + testEngineName + `

server:
  secret_key: "e2e-test-secret-key"

search:
  formats:
    - html
    - json

engines:
  - name: ` + testEngineName + `
    engine: ` + testEngineModule + `
    shortcut: ` + testEngineBang + `
    categories: general
    disabled: false
    enable_http: true

plugins:
  result_link_router.SXNGPlugin:
    active: true
`

func startSearXNG(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	pluginPath, err := filepath.Abs(pluginSourcePath)
	if err != nil {
		t.Fatalf("resolve plugin source path: %v", err)
	}

	searxng.Start(t, ctx, networkName, searxng.Config{
		Alias:        searxngAlias,
		SettingsYAML: testSettingsYAML,
		Env: map[string]string{
			"PYTHONPATH":             pluginMountDir,
			"VISITCRAWL_LINK_SECRET": visitLinkSecret,
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      pluginPath,
				ContainerFilePath: pluginMountDir + "/result_link_router.py",
				FileMode:          0o644,
			},
			{
				Reader:            strings.NewReader(testEngineSource(searchUpstreamNetworkURL())),
				ContainerFilePath: engineMountDir + "/" + testEngineModule + ".py",
				FileMode:          0o644,
			},
		},
	})
}
