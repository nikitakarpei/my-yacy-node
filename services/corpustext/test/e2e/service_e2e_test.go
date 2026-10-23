//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/crawlscrapebridgeservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagescrapeservice"
)

func TestScrapeRequestStaysSearchableInElasticsearch(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	elasticsearchURL := startElasticsearch(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	pagescrapeservice.Start(t, ctx, network.Name)
	startNode(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	crawlscrapebridgeservice.Start(t, ctx, network.Name)
	corpusText := corpustextservice.Start(
		t, ctx, network.Name, crawlNATSURL, elasticsearchCorpusTextEnv(),
	)

	js := connectJetStream(t, crawlNATSURL)
	awaitOrdersStream(t, ctx, js)

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
	assertCorpusTextFinds(t, ctx, corpusText, originURL)

	corpusText.Restart(t, ctx)

	hitAfterRestart := waitForElasticsearchContentHit(
		t, ctx, elasticsearchURL, elasticsearchIndexPattern, stemmedTerm,
	)
	assertIndexedPage(t, hitAfterRestart, originURL)
	assertCorpusTextFinds(t, ctx, corpusText, originURL)
}

func TestScrapeRequestStaysSearchableInManticore(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	manticoreURL := startManticore(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	pagescrapeservice.Start(t, ctx, network.Name)
	startNode(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	crawlscrapebridgeservice.Start(t, ctx, network.Name)
	corpusText := corpustextservice.Start(
		t, ctx, network.Name, crawlNATSURL, manticoreCorpusTextEnv(),
	)

	js := connectJetStream(t, crawlNATSURL)
	awaitOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	hit := waitForManticoreContentHit(t, ctx, manticoreURL, manticoreLanguageTable, stemmedTerm)
	assertIndexedPage(t, hit, originURL)

	fanOutHit := waitForManticoreContentHit(t, ctx, manticoreURL, manticoreFanOutTable, stemmedTerm)
	assertIndexedPage(t, fanOutHit, originURL)
	assertCorpusTextFinds(t, ctx, corpusText, originURL)

	corpusText.Restart(t, ctx)

	hitAfterRestart := waitForManticoreContentHit(
		t,
		ctx,
		manticoreURL,
		manticoreFanOutTable,
		stemmedTerm,
	)
	assertIndexedPage(t, hitAfterRestart, originURL)
	assertCorpusTextFinds(t, ctx, corpusText, originURL)
}
