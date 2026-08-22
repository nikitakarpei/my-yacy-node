//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	corpusTextAlias    = "corpustext"
	envCorpusTextImage = "CORPUSTEXT_IMAGE"
	manticoreAlias     = "manticore"
	manticoreTableBase = "yacy_text"
	indexedLanguage    = "en"
	manticoreTable     = manticoreTableBase + "_v1_" + indexedLanguage
)

func startManticore(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return manticore.Start(t, ctx, networkName, manticoreAlias)
}

func startCorpusText(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envCorpusTextImage, "corpustext", "e2e"),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {corpusTextAlias}},
			Env: map[string]string{
				"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL(),
				"CORPUSTEXT_PROXY_URL":    egressproxy.NetworkURL(),
				"SEARCH_INDEX_ENGINE":     "manticore",
				"MANTICORE_URL":           manticore.NetworkURL(manticoreAlias),
				"MANTICORE_TABLE":         manticoreTableBase,
				"CORPUSTEXT_LANGUAGES":    indexedLanguage,
				"LOG_LEVEL":               "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start corpustext container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpustext", container)
}
