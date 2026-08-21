//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
)

func TestReachedTextSearchReadsEveryLanguageIndexInElasticsearch(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	elasticsearchHostURL := startElasticsearch(t, ctx, network.Name)

	egressproxy.Start(t, ctx, network.Name)
	startOrigins(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createReachedPagesStream(t, ctx, js)

	startCorpusText(t, ctx, network.Name, elasticsearchCorpusTextEnv())
	publishReachedCorpus(t, ctx, js)
	awaitElasticsearchCorpus(t, ctx, elasticsearchHostURL)
	assertCatchAllHoldsTheUnconfiguredPage(
		t,
		documentsInElasticsearchIndex(t, ctx, elasticsearchHostURL, elasticsearchCatchAllIndex),
	)

	searxngBaseURL := startSearXNG(t, ctx, network.Name, elasticsearchEngineSettings())

	assertReachedCorpusIsSearchable(t, ctx, searxngBaseURL)
}

func TestReachedTextSearchReadsEveryLanguageTableInManticore(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	manticoreHostURL := startManticore(t, ctx, network.Name)

	egressproxy.Start(t, ctx, network.Name)
	startOrigins(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	createReachedPagesStream(t, ctx, js)

	startCorpusText(t, ctx, network.Name, manticoreCorpusTextEnv())
	publishReachedCorpus(t, ctx, js)
	awaitManticoreCorpus(t, ctx, manticoreHostURL)
	assertCatchAllHoldsTheUnconfiguredPage(
		t,
		documentsInManticoreTable(t, ctx, manticoreHostURL, manticoreCatchAllTable),
	)

	searxngBaseURL := startSearXNG(t, ctx, network.Name, manticoreEngineSettings())

	assertReachedCorpusIsSearchable(t, ctx, searxngBaseURL)
}
