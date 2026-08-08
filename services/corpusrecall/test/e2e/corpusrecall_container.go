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
	corpusRecallAlias    = "corpusrecall"
	corpusRecallPort     = "8092/tcp"
	envCorpusRecallImage = "CORPUSRECALL_IMAGE"
	corpusRecallLimit    = 5 * time.Second
)

func startCorpusRecall(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          corpusRecallImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {corpusRecallAlias}},
			ExposedPorts:   []string{corpusRecallPort},
			WaitingFor:     wait.ForListeningPort(corpusRecallPort),
			Env: map[string]string{
				"NATS_URL":                  natsjetstream.NetworkURL(),
				"CORPUSRECALL_RECALL_LIMIT": corpusRecallLimit.String(),
				"LOG_LEVEL":                 "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start corpusrecall container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpusrecall", container)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("corpusrecall host: %v", err)
	}
	port, err := container.MappedPort(ctx, corpusRecallPort)
	if err != nil {
		t.Fatalf("corpusrecall mapped port: %v", err)
	}
	return host + ":" + port.Port()
}

func corpusRecallImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCorpusRecallImage, "corpusrecall", "e2e")
}
