//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
)

func TestCrawledPageStaysSearchableInElasticsearch(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	elasticsearchURL := startElasticsearch(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startNode(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	corpusText := startCorpusText(t, ctx, network.Name, elasticsearchCorpusTextEnv())

	js := connectJetStream(t, natsURL)
	ensureOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	hit := waitForElasticsearchContentHit(
		t,
		ctx,
		elasticsearchURL,
		elasticsearchLanguageIndex,
		stemmedTerm,
	)
	assertIndexedPage(t, hit, originURL)

	fanOutHit := waitForElasticsearchContentHit(
		t, ctx, elasticsearchURL, elasticsearchIndexPattern, stemmedTerm,
	)
	assertIndexedPage(t, fanOutHit, originURL)

	restartCorpusText(t, ctx, corpusText)

	hitAfterRestart := waitForElasticsearchContentHit(
		t, ctx, elasticsearchURL, elasticsearchIndexPattern, stemmedTerm,
	)
	assertIndexedPage(t, hitAfterRestart, originURL)
}

func TestCrawledPageStaysSearchableInManticore(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	manticoreURL := startManticore(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startNode(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	corpusText := startCorpusText(t, ctx, network.Name, manticoreCorpusTextEnv())

	js := connectJetStream(t, natsURL)
	ensureOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	hit := waitForManticoreContentHit(t, ctx, manticoreURL, manticoreLanguageTable, stemmedTerm)
	assertIndexedPage(t, hit, originURL)

	fanOutHit := waitForManticoreContentHit(t, ctx, manticoreURL, manticoreFanOutTable, stemmedTerm)
	assertIndexedPage(t, fanOutHit, originURL)

	restartCorpusText(t, ctx, corpusText)

	hitAfterRestart := waitForManticoreContentHit(
		t,
		ctx,
		manticoreURL,
		manticoreFanOutTable,
		stemmedTerm,
	)
	assertIndexedPage(t, hitAfterRestart, originURL)
}
