//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	firecrawlShimAlias    = "firecrawlshim"
	firecrawlShimPort     = "8093/tcp"
	envFirecrawlShimImage = "FIRECRAWLSHIM_IMAGE"
	crawlOutcomesTarget   = crawlerAlias + ":8095"
	markdownCorpusTarget  = corpusMarkdownAlias + ":8094"
	defaultRecallLimit    = 30 * time.Second
)

func startFirecrawlShim(
	t *testing.T,
	ctx context.Context,
	networkName string,
	recallLimit time.Duration,
) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          firecrawlShimImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {firecrawlShimAlias}},
			ExposedPorts:   []string{firecrawlShimPort},
			WaitingFor:     wait.ForListeningPort(firecrawlShimPort),
			Env: map[string]string{
				"CRAWL_NATS_URL":                       natsjetstream.NetworkURL(),
				"FIRECRAWLSHIM_CRAWL_OUTCOMES_TARGET":  crawlOutcomesTarget,
				"FIRECRAWLSHIM_MARKDOWN_CORPUS_TARGET": markdownCorpusTarget,
				"FIRECRAWLSHIM_RECALL_LIMIT":           recallLimit.String(),
				"LOG_LEVEL":                            "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start firecrawlshim container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "firecrawlshim", container)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("firecrawlshim host: %v", err)
	}
	port, err := container.MappedPort(ctx, firecrawlShimPort)
	if err != nil {
		t.Fatalf("firecrawlshim mapped port: %v", err)
	}
	return "http://" + host + ":" + port.Port()
}

func firecrawlShimImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envFirecrawlShimImage, "firecrawlshim", "e2e")
}
