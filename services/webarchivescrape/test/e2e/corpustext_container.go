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

const manticoreTable = "yacy_text_v1_en"

func startCorpusText(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			Started: true,
			ContainerRequest: testcontainers.ContainerRequest{
				Image: requiredimage.FromEnv(
					t,
					"CORPUSTEXT_IMAGE",
					"corpustext",
					"e2e-webarchivescrape",
				),
				Networks: []string{networkName},
				Env: map[string]string{
					"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL(),
					"SCRAPE_PROXY_URL":        egressproxy.NetworkURL(),
					"SEARCH_INDEX_ENGINE":     "manticore",
					"MANTICORE_URL":           manticore.NetworkURL("manticore"),
					"MANTICORE_TABLE":         "yacy_text",
					"CORPUSTEXT_LANGUAGES":    "en",
					"LOG_LEVEL":               "debug",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("start corpustext container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpustext", container)
}
