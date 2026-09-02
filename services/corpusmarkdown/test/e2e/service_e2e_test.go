//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagescrapeservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/markdowncorpusclienttest"
)

const (
	storageDeadline    = 90 * time.Second
	originCanonicalURL = "http://" + originAlias + "/"
)

func TestCrawledPageMarkdownIsStoredAndRecalledByURL(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	pagescrapeservice.Start(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)
	recallAddress := startCorpusMarkdown(t, ctx, network.Name)

	js := connectJetStream(t, crawlNATSURL)
	awaitOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	store := awaitPageMarkdownBucket(t, ctx, js)
	objectName := pagemarkdownstore.ObjectNameOf(
		canonicalurltest.CanonicalURLOf(t, originCanonicalURL),
	)

	var stored []byte
	found := pollwait.For(storageDeadline, func() bool {
		markdown, err := store.GetBytes(ctx, objectName)
		if err != nil {
			return false
		}
		stored = markdown
		return true
	})
	if !found {
		t.Fatalf("markdown object %q not stored within %s", objectName, storageDeadline)
	}
	if len(stored) == 0 {
		t.Fatal("stored markdown is empty")
	}
	if !strings.Contains(string(stored), originBody) {
		t.Errorf("stored markdown = %q, want it to contain %q", stored, originBody)
	}

	recallPageResponse, err := markdowncorpusclienttest.New(t, recallAddress).
		RecallPage(ctx, &corpusmarkdownv1.RecallPageRequest{Url: originCanonicalURL})
	if err != nil {
		t.Fatalf("recall page: %v", err)
	}
	if recallPageResponse.GetMarkdown() != string(stored) {
		t.Errorf(
			"recalled markdown = %q, want the stored %q",
			recallPageResponse.GetMarkdown(), stored,
		)
	}
	if recallPageResponse.GetCanonicalUrl() != originCanonicalURL {
		t.Errorf(
			"recalled canonicalUrl = %q, want %q",
			recallPageResponse.GetCanonicalUrl(), originCanonicalURL,
		)
	}
}
