//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	corpusMarkdownAlias    = "corpusmarkdown"
	envCorpusMarkdownImage = "CORPUSMARKDOWN_IMAGE"
)

func startCorpusMarkdown(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          corpusMarkdownImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {corpusMarkdownAlias}},
			Env: map[string]string{
				"CRAWL_NATS_URL":            natsjetstream.NetworkURL(),
				"PAGE_MARKDOWN_NATS_URL":    natsjetstream.NetworkURL(),
				"NATS_CRAWLED_PAGE_SUBJECT": crawledPageSubject,
				"LOG_LEVEL":                 "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start corpusmarkdown container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpusmarkdown", container)
}

func corpusMarkdownImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCorpusMarkdownImage, "corpusmarkdown", "e2e")
}
