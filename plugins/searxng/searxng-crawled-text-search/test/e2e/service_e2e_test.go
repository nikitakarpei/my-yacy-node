//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
)

func TestCrawledTextSearchReadsEveryLanguageIndexInElasticsearch(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	elasticsearchHostURL := startElasticsearch(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createCrawledPageStream(t, ctx, js)

	startCorpusText(t, ctx, network.Name, elasticsearchCorpusTextEnv())
	publishCrawledCorpus(t, ctx, js)
	awaitElasticsearchCorpus(t, ctx, elasticsearchHostURL)
	assertCatchAllHoldsTheUnconfiguredPage(
		t,
		documentsInElasticsearchIndex(t, ctx, elasticsearchHostURL, elasticsearchCatchAllIndex),
	)

	searxngBaseURL := startSearXNG(t, ctx, network.Name, elasticsearchEngineSettings())

	assertCrawledCorpusIsSearchable(t, ctx, searxngBaseURL)
}

func TestCrawledTextSearchReadsEveryLanguageTableInManticore(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	manticoreHostURL := startManticore(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createCrawledPageStream(t, ctx, js)

	startCorpusText(t, ctx, network.Name, manticoreCorpusTextEnv())
	publishCrawledCorpus(t, ctx, js)
	awaitManticoreCorpus(t, ctx, manticoreHostURL)
	assertCatchAllHoldsTheUnconfiguredPage(
		t,
		documentsInManticoreTable(t, ctx, manticoreHostURL, manticoreCatchAllTable),
	)

	searxngBaseURL := startSearXNG(t, ctx, network.Name, manticoreEngineSettings())

	assertCrawledCorpusIsSearchable(t, ctx, searxngBaseURL)
}
