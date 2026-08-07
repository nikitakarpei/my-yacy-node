//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
)

func TestCrawledPageIsSearchableInElasticsearch(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	elasticsearchURL := startElasticsearch(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startNode(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	startCorpusText(t, ctx, network.Name, elasticsearchCorpusTextEnv())

	js := connectJetStream(t, natsURL)
	ensureOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	hit := waitForElasticsearchContentHit(t, ctx, elasticsearchURL, languageIndexName, stemmedTerm)
	assertIndexedPage(t, hit, originURL)

	fanOutHit := waitForElasticsearchContentHit(
		t, ctx, elasticsearchURL, fanOutPattern, stemmedTerm,
	)
	assertIndexedPage(t, fanOutHit, originURL)
}

func TestCrawledPageIsSearchableInManticore(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	manticoreURL := startManticore(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startNode(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	startCorpusText(t, ctx, network.Name, manticoreCorpusTextEnv())

	js := connectJetStream(t, natsURL)
	ensureOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	hit := waitForManticoreContentHit(t, ctx, manticoreURL, languageIndexName, stemmedTerm)
	assertIndexedPage(t, hit, originURL)

	fanOutHit := waitForManticoreContentHit(t, ctx, manticoreURL, fanOutPrefix, stemmedTerm)
	assertIndexedPage(t, fanOutHit, originURL)
}

func assertIndexedPage(t *testing.T, hit searchHit, originURL string) {
	t.Helper()
	if hit.Source.Title != originTitle {
		t.Errorf("indexed title = %q, want %q", hit.Source.Title, originTitle)
	}
	if !strings.Contains(hit.Source.Content, originBody) {
		t.Errorf("indexed content = %q, want it to contain %q", hit.Source.Content, originBody)
	}
	if hit.Source.Language != indexedLanguage {
		t.Errorf("indexed language = %q, want %q", hit.Source.Language, indexedLanguage)
	}
	if !strings.Contains(hit.Source.URL, strings.TrimSuffix(originURL, "/")) {
		t.Errorf("indexed url = %q, want it to carry %q", hit.Source.URL, originURL)
	}
}
