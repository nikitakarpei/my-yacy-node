//go:build e2e

// Package pagerwiservice starts the service that builds the reverse word index of each
// offered page and publishes it for the nodes. It owns the RWI stream, so a suite that
// starts it provisions nothing itself.
package pagerwiservice

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
	alias          = "pagerwi"
	opsPort        = "9090/tcp"
	opsPathMetrics = "/metrics"
	envImage       = "PAGERWI_IMAGE"
)

func Start(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envImage, "pagerwi", "e2e"),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			ExposedPorts:   []string{opsPort},
			WaitingFor: wait.ForHTTP(opsPathMetrics).
				WithPort(opsPort).
				WithForcedIPv4LocalHost(),
			Env: map[string]string{
				"SCRAPE_PAGE_OFFER_NATS_URL": natsjetstream.NetworkURL(),
				"PAGE_RWI_NATS_URL":          natsjetstream.NetworkURL(),
				"LOG_LEVEL":                  "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start pagerwi container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "pagerwi", container)
}
