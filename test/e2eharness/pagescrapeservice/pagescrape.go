//go:build e2e

// Package pagescrapeservice starts the service that reads each requested page once and
// offers it to the corpora. It owns the scrape streams, so a suite that starts it
// provisions nothing itself.
package pagescrapeservice

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
	alias          = "pagescrape"
	opsPort        = "9090/tcp"
	opsPathMetrics = "/metrics"
	envImage       = "PAGESCRAPE_IMAGE"
)

func Start(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envImage, "pagescrape", "e2e"),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			ExposedPorts:   []string{opsPort},
			WaitingFor: wait.ForHTTP(opsPathMetrics).
				WithPort(opsPort).
				WithForcedIPv4LocalHost(),
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
