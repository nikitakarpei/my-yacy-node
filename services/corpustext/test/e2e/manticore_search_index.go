//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
)

const (
	manticoreAlias         = "manticore"
	manticoreTableBase     = "yacy_text"
	manticoreFanOutTable   = manticoreTableBase + "_v1"
	manticoreLanguageTable = manticoreFanOutTable + "_" + indexedLanguage
)

func startManticore(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return manticore.Start(t, ctx, networkName, manticoreAlias)
}

func manticoreNetworkURL() string {
	return manticore.NetworkURL(manticoreAlias)
}

func manticoreCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE":  "manticore",
		"MANTICORE_URL":        manticoreNetworkURL(),
		"MANTICORE_TABLE":      manticoreTableBase,
		"CORPUSTEXT_LANGUAGES": indexedLanguage,
	}
}
