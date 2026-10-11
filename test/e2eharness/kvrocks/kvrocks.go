//go:build e2e

// Package kvrocks starts a disposable Apache Kvrocks container that serves the Redis
// protocol for e2e suites.
package kvrocks

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	image = "docker.io/apache/kvrocks:2.17.0"
	alias = "kvrocks"
	port  = "6666/tcp"
)

func Start(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image,
			ExposedPorts:   []string{port},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			WaitingFor:     wait.ForListeningPort(port).WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start Kvrocks container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, alias, container)
}

func NetworkURL() string {
	return "redis://" + alias + ":6666"
}
