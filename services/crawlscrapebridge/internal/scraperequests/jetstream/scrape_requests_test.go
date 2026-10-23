package jetstream_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	scraperequestsjetstream "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/internal/scraperequests/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	pageURL        = "https://example.org/page"
	publishTimeout = 2 * time.Second
)

type recordingObserver struct {
	mu           sync.Mutex
	published    []canonicalurl.CanonicalURL
	notPublished []canonicalurl.CanonicalURL
}

func (o *recordingObserver) ScrapeRequestPublished(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published = append(o.published, pageURL)
}

func (o *recordingObserver) ScrapeRequestNotPublished(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ error,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.notPublished = append(o.notPublished, pageURL)
}

func TestAskToScrapePublishesAScrapeRequestForThePage(t *testing.T) {
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := broker.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     pagescrapecontract.ScrapeRequestsStreamName,
		Subjects: []string{pagescrapecontract.ScrapeRequestSubject},
	})
	if err != nil {
		t.Fatalf("create the scrape requests stream: %v", err)
	}
	observer := &recordingObserver{}
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	if !scraperequestsjetstream.NewScrapeRequests(broker, observer).
		AskToScrape(context.Background(), page) {
		t.Fatal("AskToScrape = false, want the request published")
	}

	message, err := stream.GetLastMsgForSubject(
		context.Background(), pagescrapecontract.ScrapeRequestSubject,
	)
	if err != nil {
		t.Fatalf("read the scrape request: %v", err)
	}
	request, err := pagescrapecontract.UnmarshalScrapeRequest(message.Data)
	if err != nil {
		t.Fatalf("unmarshal the scrape request: %v", err)
	}
	if request.PageURL != page {
		t.Errorf("scrape request page = %s, want %s", request.PageURL, page)
	}
	if len(observer.published) != 1 || len(observer.notPublished) != 0 {
		t.Errorf("published %v, not published %v", observer.published, observer.notPublished)
	}
}

func TestAskToScrapeReportsARequestThatFindsNoStream(t *testing.T) {
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	observer := &recordingObserver{}
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	if scraperequestsjetstream.NewScrapeRequests(broker, observer).
		AskToScrape(ctx, canonicalurltest.CanonicalURLOf(t, pageURL)) {
		t.Fatal("AskToScrape = true, want the request not published")
	}

	if len(observer.notPublished) != 1 || len(observer.published) != 0 {
		t.Errorf("published %v, not published %v", observer.published, observer.notPublished)
	}
}
