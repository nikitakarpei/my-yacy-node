//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	manticoreTable    = "yacy_text_v1_en"
	corpusTextDurable = "corpustext"

	corpusTextDurableAppearance = 60 * time.Second
)

func startCorpusText(t *testing.T, ctx context.Context, networkName, natsURL string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			Started: true,
			ContainerRequest: testcontainers.ContainerRequest{
				Image: requiredimage.FromEnv(
					t,
					"CORPUSTEXT_IMAGE",
					"corpustext",
					"e2e-webarchivescrape",
				),
				Networks: []string{networkName},
				Env: map[string]string{
					"SCRAPE_PAGE_OFFER_NATS_URL": natsjetstream.NetworkURL(),
					"SEARCH_INDEX_ENGINE":        "manticore",
					"MANTICORE_URL":              manticore.NetworkURL("manticore"),
					"MANTICORE_TABLE":            "yacy_text",
					"CORPUSTEXT_LANGUAGES":       "en",
					"LOG_LEVEL":                  "debug",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("start corpustext container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "corpustext", container)
	awaitCorpusTextDurable(t, ctx, jetStreamAt(t, natsURL))
}

func jetStreamAt(t *testing.T, natsURL string) jetstream.JetStream {
	t.Helper()
	connection, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(connection.Close)
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func awaitCorpusTextDurable(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	registered := pollwait.For(corpusTextDurableAppearance, func() bool {
		stream, err := js.Stream(ctx, pagescrapecontract.ScrapePageOffersStreamName)
		if err != nil {
			return false
		}
		_, err = stream.Consumer(ctx, corpusTextDurable)
		return err == nil
	})
	if !registered {
		t.Fatalf("corpustext did not register its %q durable within %s",
			corpusTextDurable, corpusTextDurableAppearance)
	}
}
