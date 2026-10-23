package crawledpagerelay_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/crawledpagerelay"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	crawledPageURL   = "https://example.org/page"
	relayConcurrency = 2
)

type recordingScrapeRequests struct {
	mu       sync.Mutex
	asked    []canonicalurl.CanonicalURL
	accepted bool
}

func (r *recordingScrapeRequests) AskToScrape(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asked = append(r.asked, pageURL)
	return r.accepted
}

func crawledPage(t *testing.T) *pullintaketest.Message {
	t.Helper()
	data, err := yacycrawlcontract.MarshalCrawledPage(yacycrawlcontract.CrawledPage{
		PageURL: canonicalurltest.CanonicalURLOf(t, crawledPageURL),
	})
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	return &pullintaketest.Message{Body: data}
}

func relayed(
	t *testing.T,
	scrapeRequests *recordingScrapeRequests,
	message *pullintaketest.Message,
) error {
	t.Helper()
	return crawledpagerelay.NewCrawledPageRelay(scrapeRequests, relayConcurrency).Run(
		context.Background(),
		pullintake.PendingMessagesOf(pullintaketest.MessageSourceOf(message)),
	)
}

func TestACrawledPageIsAskedToBeScrapedAndAcknowledged(t *testing.T) {
	scrapeRequests := &recordingScrapeRequests{accepted: true}
	message := crawledPage(t)

	if err := relayed(t, scrapeRequests, message); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, crawledPageURL)}
	if !slices.Equal(scrapeRequests.asked, want) {
		t.Errorf("asked to scrape %v, want %v", scrapeRequests.asked, want)
	}
	if settlement := message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("crawled page was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestACrawledPageWhoseScrapeRequestDidNotLeaveIsHeldBack(t *testing.T) {
	scrapeRequests := &recordingScrapeRequests{accepted: false}
	message := crawledPage(t)

	if err := relayed(t, scrapeRequests, message); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if settlement := message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("crawled page was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
	if message.HeldBackFor() != pullintake.ReturnPause {
		t.Errorf("held back for %s, want %s", message.HeldBackFor(), pullintake.ReturnPause)
	}
}

func TestACrawledPageThatCannotBeReadHaltsTheRelay(t *testing.T) {
	scrapeRequests := &recordingScrapeRequests{accepted: true}
	message := &pullintaketest.Message{Body: []byte("not json")}

	err := relayed(t, scrapeRequests, message)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Errorf("Run = %v, want the poison halt", err)
	}
	if len(scrapeRequests.asked) != 0 {
		t.Errorf("asked to scrape %v, want nothing", scrapeRequests.asked)
	}
	if settlements := message.Settlements(); len(settlements) != 0 {
		t.Errorf("unreadable page was settled %v, want it left pending", settlements)
	}
}
