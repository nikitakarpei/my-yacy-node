//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
)

const (
	manticoreAlias  = "manticore"
	indexedLanguage = "en"
)

func startManticore(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	manticore.Start(t, ctx, networkName, manticoreAlias)
}

func manticoreCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE":  "manticore",
		"MANTICORE_URL":        manticore.NetworkURL(manticoreAlias),
		"CORPUSTEXT_LANGUAGES": indexedLanguage,
	}
}
