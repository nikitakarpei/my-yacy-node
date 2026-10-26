//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
)

const indexedLanguages = englishLanguage + "," + germanLanguage

func startCorpusText(
	t *testing.T,
	ctx context.Context,
	networkName, natsURL string,
) corpustextservice.CorpusText {
	t.Helper()
	return corpustextservice.Start(
		t, ctx, networkName, natsURL,
		map[string]string{"CORPUSTEXT_LANGUAGES": indexedLanguages},
	)
}
