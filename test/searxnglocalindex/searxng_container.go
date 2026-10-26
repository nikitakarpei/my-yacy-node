//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxng"
)

const (
	searxngAlias        = "searxng"
	engineBang          = "ct"
	searxngSettingsPath = "../../doc/build-your-own-search-stack/building-blocks/searxng/" +
		"settings-local-index-and-yacy-network.yml"
)

func startSearXNG(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	settings, err := os.ReadFile(searxngSettingsPath)
	if err != nil {
		t.Fatalf("read searxng settings: %v", err)
	}

	return searxng.Start(t, ctx, networkName, searxng.Config{
		Alias:        searxngAlias,
		SettingsYAML: string(settings),
		Env: map[string]string{ //nolint:gosec // a disposable test container
			"SEARXNG_SECRET": "e2e-test-secret-key",
		},
	})
}
