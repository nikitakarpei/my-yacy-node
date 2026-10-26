//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagescrapeservice"
)

func TestLocalIndexSettingsSearchTheScrapedCorpus(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)

	egressproxy.Start(t, ctx, network.Name)
	startOrigins(t, ctx, network.Name)

	pagescrapeservice.Start(t, ctx, network.Name)
	js := connectJetStream(t, natsURL)

	corpusText := startCorpusText(t, ctx, network.Name, natsURL)
	publishScrapedCorpus(t, ctx, js)
	awaitIndexedCorpus(t, ctx, corpusText)

	searxngBaseURL := startSearXNG(t, ctx, network.Name)

	assertScrapedCorpusIsSearchable(t, ctx, searxngBaseURL)
}
