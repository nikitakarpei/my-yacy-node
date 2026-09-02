//go:build e2e

package natsjetstream

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	image = "docker.io/library/nats:2.14.2"
	alias = "nats"
	port  = "4222/tcp"

	configPath = "/etc/nats/nats-server.conf"
	config     = "jetstream: enabled\nmax_payload: 8MB\n"
)

func Start(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image: image,
			Cmd:   []string{"--config", configPath},
			Files: []testcontainers.ContainerFile{{
				Reader:            strings.NewReader(config),
				ContainerFilePath: configPath,
				FileMode:          0o644,
			}},
			ExposedPorts:   []string{port},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			WaitingFor:     wait.ForLog("Server is ready").WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start NATS container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "nats", container)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("resolve NATS host: %v", err)
	}
	mappedPort, err := container.MappedPort(ctx, port)
	if err != nil {
		t.Fatalf("resolve NATS mapped port: %v", err)
	}
	return "nats://" + net.JoinHostPort(host, mappedPort.Port())
}

func NetworkURL() string {
	return "nats://" + alias + ":4222"
}
