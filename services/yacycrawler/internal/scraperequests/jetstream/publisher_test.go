package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/scraperequests/jetstream"
)

func TestPublishWritesContractMessage(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, natsjetstream.StreamConfig{
		Name:      pagescrapecontract.ScrapeRequestsStreamName,
		Subjects:  []string{pagescrapecontract.ScrapeRequestSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	observer := &recordingScrapeRequestPublicationObserver{}
	publisher := jetstream.New(js, observer)

	const url = "http://example.com/a"
	pageURL := canonicalurltest.CanonicalURLOf(t, url)
	publisher.Publish(ctx, pageURL)

	consumer, err := js.CreateOrUpdateConsumer(ctx, pagescrapecontract.ScrapeRequestsStreamName,
		natsjetstream.ConsumerConfig{AckPolicy: natsjetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(natsjetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	_ = msg.Ack()

	page, err := pagescrapecontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if page.PageURL != pageURL {
		t.Fatalf("page url = %q, want %q", page.PageURL, url)
	}
	if page.FetchURL != pageURL {
		t.Fatalf("fetch url = %q, want the page url %q", page.FetchURL, url)
	}
	if observer.published != 1 {
		t.Fatalf("published requests = %d, want 1", observer.published)
	}
}

func TestARequestThatNeverLeavesIsObserved(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	observer := &recordingScrapeRequestPublicationObserver{}
	publisher := jetstream.New(js, observer)

	publisher.Publish(
		context.Background(), canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)

	if observer.publishingFailures != 1 {
		t.Fatalf("publishing failures = %d, want 1", observer.publishingFailures)
	}
	if observer.published != 0 {
		t.Fatalf("published requests = %d, want 0", observer.published)
	}
}

type recordingScrapeRequestPublicationObserver struct {
	published          int
	marshalingFailures int
	publishingFailures int
}

func (o *recordingScrapeRequestPublicationObserver) ScrapeRequestPublished(
	context.Context, canonicalurl.CanonicalURL,
) {
	o.published++
}

func (o *recordingScrapeRequestPublicationObserver) ScrapeRequestMarshalingFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.marshalingFailures++
}

func (o *recordingScrapeRequestPublicationObserver) ScrapeRequestPublishingFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.publishingFailures++
}
