//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	corpusTextAlias          = "corpustext"
	corpusTextOperationsPort = "9090/tcp"
	corpusTextSearchURL      = "http://" + corpusTextAlias + ":8096"
	envCorpusTextImage       = "CORPUSTEXT_IMAGE"
	indexedLanguages         = englishLanguage + "," + germanLanguage
	corpusTextDurable        = "corpustext"
	durableAppearance        = 60 * time.Second
)

func startCorpusText(
	t *testing.T,
	ctx context.Context,
	networkName string,
	searchIndexEnv map[string]string,
) {
	t.Helper()
	env := map[string]string{
		"SCRAPE_PAGE_OFFER_NATS_URL": natsjetstream.NetworkURL(),
		"CORPUSTEXT_LANGUAGES":       indexedLanguages,
		"LOG_LEVEL":                  "debug",
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
			ExposedPorts:   []string{corpusTextOperationsPort},
			WaitingFor:     wait.ForListeningPort(corpusTextOperationsPort),
			Env:            env,
		},
	})
	if err != nil {
		t.Fatalf("start corpustext container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpustext", container)
}

func awaitCorpusTextDurable(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	registered := pollwait.For(durableAppearance, func() bool {
		stream, err := js.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
		if err != nil {
			return false
		}
		_, err = stream.Consumer(ctx, corpusTextDurable)
		return err == nil
	})
	if !registered {
		t.Fatalf("corpustext did not register its %q durable within %s",
			corpusTextDurable, durableAppearance)
	}
}
