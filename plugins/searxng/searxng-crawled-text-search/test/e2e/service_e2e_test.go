//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
)

func TestScrapedCorpusTextSearchReadsEveryLanguageIndexInElasticsearch(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	elasticsearchHostURL := startElasticsearch(t, ctx, network.Name)

	egressproxy.Start(t, ctx, network.Name)
	startOrigins(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createScrapeRequestsStream(t, ctx, js)

	startCorpusText(t, ctx, network.Name, elasticsearchCorpusTextEnv())
	publishScrapedCorpus(t, ctx, js)
	awaitElasticsearchCorpus(t, ctx, elasticsearchHostURL)
	assertCatchAllHoldsTheUnconfiguredPage(
		t,
		documentsInElasticsearchIndex(t, ctx, elasticsearchHostURL, elasticsearchCatchAllIndex),
	)

	searxngBaseURL := startSearXNG(t, ctx, network.Name, elasticsearchEngineSettings())

	assertScrapedCorpusIsSearchable(t, ctx, searxngBaseURL)
}

func TestScrapedCorpusTextSearchReadsEveryLanguageTableInManticore(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	manticoreHostURL := startManticore(t, ctx, network.Name)

	egressproxy.Start(t, ctx, network.Name)
	startOrigins(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createScrapeRequestsStream(t, ctx, js)

	startCorpusText(t, ctx, network.Name, manticoreCorpusTextEnv())
	publishScrapedCorpus(t, ctx, js)
	awaitManticoreCorpus(t, ctx, manticoreHostURL)
	assertCatchAllHoldsTheUnconfiguredPage(
		t,
		documentsInManticoreTable(t, ctx, manticoreHostURL, manticoreCatchAllTable),
	)

	searxngBaseURL := startSearXNG(t, ctx, network.Name, manticoreEngineSettings())

	assertScrapedCorpusIsSearchable(t, ctx, searxngBaseURL)
}
