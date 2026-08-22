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
	crawlerAlias    = "crawler"
	crawlerPort     = "8095/tcp"
	envCrawlerImage = "YACYCRAWLER_IMAGE"
)

func startCrawler(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          crawlerImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {crawlerAlias}},
			ExposedPorts:   []string{crawlerPort},
			WaitingFor:     wait.ForListeningPort(crawlerPort),
			Env: map[string]string{
				"CRAWL_NATS_URL":                natsjetstream.NetworkURL(),
				"YACYCRAWLER_PROXY_URL":         egressproxy.NetworkURL(),
				"YACYCRAWLER_FETCH_CONCURRENCY": "1",
				"LOG_LEVEL":                     "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start crawler container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "crawler", container)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("crawler host: %v", err)
	}
	port, err := container.MappedPort(ctx, crawlerPort)
	if err != nil {
		t.Fatalf("crawler mapped port: %v", err)
	}
	return host + ":" + port.Port()
}

func crawlerImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCrawlerImage, "crawler", "e2e")
}
