//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	corpusTextAlias    = "corpustext"
	envCorpusTextImage = "CORPUSTEXT_IMAGE"
)

func startCorpusText(
	t *testing.T,
	ctx context.Context,
	networkName string,
	searchIndexEnv map[string]string,
) {
	t.Helper()
	env := map[string]string{
		"NATS_URL":                  natsjetstream.NetworkURL(),
		"NATS_CRAWLED_PAGE_SUBJECT": crawledPageSubject,
	}
	for key, value := range searchIndexEnv {
		env[key] = value
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          corpusTextImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {corpusTextAlias}},
			Env:            env,
		},
	})
	if err != nil {
		t.Fatalf("start corpustext container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpustext", container)
}

func corpusTextImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCorpusTextImage, "corpustext", "e2e")
}
