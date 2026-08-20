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
	indexedLanguages   = englishLanguage + "," + germanLanguage
)

func startCorpusText(
	t *testing.T,
	ctx context.Context,
	networkName string,
	searchIndexEnv map[string]string,
) {
	t.Helper()
	env := map[string]string{
		"CRAWL_NATS_URL":       natsjetstream.NetworkURL(),
		"CORPUSTEXT_LANGUAGES": indexedLanguages,
	}
	for key, value := range searchIndexEnv {
		env[key] = value
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          requiredimage.FromEnv(t, envCorpusTextImage, "corpustext", "e2e"),
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
