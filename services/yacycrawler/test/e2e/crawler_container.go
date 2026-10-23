//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	crawlerAlias    = "crawler"
	envCrawlerImage = "YACYCRAWLER_IMAGE"
)

func startCrawlers(t *testing.T, ctx context.Context, networkName string, count int) {
	t.Helper()
	for instance := range count {
		startCrawlerNamed(t, ctx, networkName, fmt.Sprintf("%s-%d", crawlerAlias, instance))
	}
}

func startCrawlerNamed(t *testing.T, ctx context.Context, networkName string, alias string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          crawlerImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			Env: map[string]string{
				"CRAWL_NATS_URL":                natsjetstream.NetworkURL(),
				"YACYCRAWLER_FETCH_PROXY_URL":   egressproxy.NetworkURL(),
				"YACYCRAWLER_FETCH_CONCURRENCY": "1",
				"LOG_LEVEL":                     "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start crawler container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, alias, container)
}

func crawlerImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCrawlerImage, "crawler", "e2e")
}
