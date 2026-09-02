package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	scrapeoutcomefeeds "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/scrapeoutcomefeeds/nats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	pageURL      = "https://example.org/a"
	outcomeWait  = 5 * time.Second
	carrierStart = 100 * time.Millisecond
)

func subscribeToThePageFeed(t *testing.T, connection *nats.Conn) *nats.Subscription {
	t.Helper()
	subscription, err := connection.SubscribeSync(
		pagescrapecontract.ScrapeOutcomeSubjectsOf(canonicalurltest.CanonicalURLOf(t, pageURL)),
	)
	if err != nil {
		t.Fatalf("subscribe to the page feed: %v", err)
	}
	t.Cleanup(func() { _ = subscription.Unsubscribe() })
	return subscription
}

func TestScrapeFailureReachesAReaderOfThePageFeed(t *testing.T) {
	connection := natstestserver.Connect(t, natstestserver.Start(t))
	subscription := subscribeToThePageFeed(t, connection)
	failure := pagescrapecontract.ScrapeFailure{
		PageURL:  canonicalurltest.CanonicalURLOf(t, pageURL),
		FetchURL: canonicalurltest.CanonicalURLOf(t, pageURL),
		Reason:   pagescrapecontract.NoReasonGiven,
	}

	if err := scrapeoutcomefeeds.NewScrapeOutcomeFeed(connection).
		AnnounceScrapeFailure(context.Background(), failure); err != nil {
		t.Fatalf("announce the scrape failure: %v", err)
	}

	message, err := subscription.NextMsg(outcomeWait)
	if err != nil {
		t.Fatalf("read the page feed: %v", err)
	}
	want := pagescrapecontract.ScrapeFailureOutcomeSubjectOf(failure.PageURL)
	if message.Subject != want {
		t.Errorf("failure announced on %q, want %q", message.Subject, want)
	}
	announced, err := pagescrapecontract.UnmarshalScrapeFailure(message.Data)
	if err != nil {
		t.Fatalf("unmarshal the announced failure: %v", err)
	}
	if announced != failure {
		t.Errorf("announced %#v, want %#v", announced, failure)
	}
}

func TestIntakeReceiptIsCarriedOntoThePageFeedAsItStands(t *testing.T) {
	connection := natstestserver.Connect(t, natstestserver.Start(t))
	subscription := subscribeToThePageFeed(t, connection)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go func() {
		_ = scrapeoutcomefeeds.NewScrapeOutcomeFeed(connection).CarryIntakeReceipts(ctx)
	}()
	time.Sleep(carrierStart)

	pageCanonicalURL := canonicalurltest.CanonicalURLOf(t, pageURL)
	receipt, err := pagescrapecontract.MarshalKeptPage(pagescrapecontract.KeptPage{
		PageURL: pageCanonicalURL,
		Corpus:  "corpusmarkdown",
	})
	if err != nil {
		t.Fatalf("marshal the receipt: %v", err)
	}
	if err := connection.Publish(
		pagescrapecontract.KeptPageSubjectOf(pageCanonicalURL), receipt,
	); err != nil {
		t.Fatalf("send the receipt: %v", err)
	}

	message, err := subscription.NextMsg(outcomeWait)
	if err != nil {
		t.Fatalf("read the page feed: %v", err)
	}
	want := pagescrapecontract.KeptPageOutcomeSubjectOf(pageCanonicalURL)
	if message.Subject != want {
		t.Errorf("receipt carried onto %q, want %q", message.Subject, want)
	}
	if string(message.Data) != string(receipt) {
		t.Errorf("carried %q, want the receipt %q as it stands", message.Data, receipt)
	}
}
