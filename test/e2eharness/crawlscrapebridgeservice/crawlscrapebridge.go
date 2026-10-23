//go:build e2e

// Package crawlscrapebridgeservice starts the service that asks the scrape service to scrape
// each page the crawler found indexable. It waits for the crawler's stream itself, so a suite
// may start it before the crawler.
package crawlscrapebridgeservice

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
	alias          = "crawlscrapebridge"
	opsPort        = "9090/tcp"
	opsPathMetrics = "/metrics"
	envImage       = "CRAWLSCRAPEBRIDGE_IMAGE"
)

func Start(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envImage, "crawlscrapebridge", "e2e"),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			ExposedPorts:   []string{opsPort},
			WaitingFor: wait.ForHTTP(opsPathMetrics).
				WithPort(opsPort).
				WithForcedIPv4LocalHost(),
			Env: map[string]string{
				"CRAWL_NATS_URL":          natsjetstream.NetworkURL(),
				"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL(),
				"LOG_LEVEL":               "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start crawlscrapebridge container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "crawlscrapebridge", container)
}
