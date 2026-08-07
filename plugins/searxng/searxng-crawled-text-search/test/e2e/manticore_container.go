//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
)

const manticoreAlias = "manticore"

func startManticore(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return manticore.Start(t, ctx, networkName, manticoreAlias)
}

func awaitManticoreCorpus(t *testing.T, ctx context.Context, manticoreURL string) {
	t.Helper()
	awaitIndexedCorpus(
		t,
		ctx,
		manticoreURL+"/search",
		map[string]any{
			"table": fanOutPrefix,
			"query": map[string]any{"match_all": map[string]any{}},
		},
	)
}

func manticoreCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE": "manticore",
		"MANTICORE_URL":       manticore.NetworkURL(manticoreAlias),
		"MANTICORE_TABLE":     indexBaseName,
	}
}

func manticoreEngineSettings() string {
	return "    search_index_engine: manticore\n" +
		"    manticore_url: " + manticore.NetworkURL(manticoreAlias) + "\n" +
		"    manticore_table: " + fanOutPrefix + "\n"
}
