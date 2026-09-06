//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	corpusMarkdownAlias          = "corpusmarkdown"
	corpusMarkdownOperationsPort = "9090/tcp"
	corpusMarkdownMetricsPath    = "/metrics"
	envCorpusMarkdownImage       = "CORPUSMARKDOWN_IMAGE"
)

func startCorpusMarkdown(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image: requiredimage.FromEnv(
				t,
				envCorpusMarkdownImage,
				"corpusmarkdown",
				"e2e",
			),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {corpusMarkdownAlias}},
			ExposedPorts:   []string{corpusMarkdownOperationsPort},
			WaitingFor: wait.ForHTTP(corpusMarkdownMetricsPath).
				WithPort(corpusMarkdownOperationsPort).
				WithForcedIPv4LocalHost(),
			Env: map[string]string{
				"SCRAPE_PAGE_OFFER_NATS_URL": natsjetstream.NetworkURL(),
				"PAGE_MARKDOWN_NATS_URL":     natsjetstream.NetworkURL(),
				"LOG_LEVEL":                  "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start corpusmarkdown container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpusmarkdown", container)
}
