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
	corpusMarkdownAlias    = "corpusmarkdown"
	corpusMarkdownPort     = "8094/tcp"
	envCorpusMarkdownImage = "CORPUSMARKDOWN_IMAGE"
)

func startCorpusMarkdown(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          corpusMarkdownImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {corpusMarkdownAlias}},
			ExposedPorts:   []string{corpusMarkdownPort},
			WaitingFor:     wait.ForListeningPort(corpusMarkdownPort),
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

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("corpusmarkdown host: %v", err)
	}
	port, err := container.MappedPort(ctx, corpusMarkdownPort)
	if err != nil {
		t.Fatalf("corpusmarkdown mapped port: %v", err)
	}
	return host + ":" + port.Port()
}

func corpusMarkdownImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCorpusMarkdownImage, "corpusmarkdown", "e2e")
}
