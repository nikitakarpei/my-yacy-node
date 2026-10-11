//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pagescrapeservice"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/scraperequestbridge"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
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
	scraperequestbridge.Relay(t, ctx, crawlNATSURL)
	recallAddress := startCorpusMarkdown(t, ctx, network.Name)

	js := connectJetStream(t, crawlNATSURL)
	awaitOrdersStream(t, ctx, js)

	publishCrawlOrder(t, ctx, js, originURL)

	client := markdowncorpusclienttest.New(t, recallAddress)
	var recalled pagemarkdownstore.RecalledPage
	found := pollwait.For(storageDeadline, func() bool {
		page, statusCode := client.RecallPage(ctx, originCanonicalURL)
		recalled = page
		return statusCode == http.StatusOK
	})
	if !found {
		t.Fatalf("markdown of %q not recalled within %s", originCanonicalURL, storageDeadline)
	}
	if !strings.Contains(recalled.Markdown, originBody) {
		t.Errorf("recalled markdown = %q, want it to contain %q", recalled.Markdown, originBody)
	}
	if recalled.CanonicalURL != originCanonicalURL {
		t.Errorf("recalled canonicalUrl = %q, want %q", recalled.CanonicalURL, originCanonicalURL)
	}
}
