//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	pageScrapeAlias    = "pagescrape"
	pageScrapeOpsPort  = "9090/tcp"
	envPageScrapeImage = "PAGESCRAPE_IMAGE"
)

func startPageScrape(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envPageScrapeImage, "pagescrape", "e2e"),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {pageScrapeAlias}},
			ExposedPorts:   []string{pageScrapeOpsPort},
			WaitingFor:     wait.ForListeningPort(pageScrapeOpsPort),
			Env: map[string]string{
				"SCRAPE_NATS_URL":  natsjetstream.NetworkURL(),
				"SCRAPE_PROXY_URL": egressproxy.NetworkURL(),
				"LOG_LEVEL":        "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start pagescrape container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "pagescrape", container)
}
