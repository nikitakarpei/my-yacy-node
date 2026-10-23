//go:build e2e

package e2e

import (
	"context"
	"maps"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
)

const indexedLanguages = englishLanguage + "," + germanLanguage

func startCorpusText(
	t *testing.T,
	ctx context.Context,
	networkName, natsURL string,
	searchIndexEnv map[string]string,
) corpustextservice.CorpusText {
	t.Helper()
	env := map[string]string{"CORPUSTEXT_LANGUAGES": indexedLanguages}
	maps.Copy(env, searchIndexEnv)
	return corpustextservice.Start(t, ctx, networkName, natsURL, env)
}
