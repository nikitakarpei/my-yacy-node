//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	corpusTextAlias       = "corpustext"
	envCorpusTextImage    = "CORPUSTEXT_IMAGE"
	indexedLanguage       = "en"
	corpusTextStopTimeout = 30 * time.Second
)

func startCorpusText(
	t *testing.T,
	ctx context.Context,
	networkName string,
	searchIndexEnv map[string]string,
) testcontainers.Container {
	t.Helper()
	env := map[string]string{
		"SCRAPE_PAGE_OFFER_NATS_URL": natsjetstream.NetworkURL(),
		"LOG_LEVEL":                  "debug",
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
	return container
}

func restartCorpusText(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()
	stopTimeout := corpusTextStopTimeout
	if err := container.Stop(ctx, &stopTimeout); err != nil {
		t.Fatalf("stop corpustext container: %v", err)
	}
	if err := container.Start(ctx); err != nil {
		t.Fatalf("restart corpustext container: %v", err)
	}
}

func corpusTextImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envCorpusTextImage, "corpustext", "e2e")
}
