package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

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
	publisher := jetstream.New(js)

	const url = "http://example.com/a"
	pageURL := canonicalurltest.CanonicalURLOf(t, url)
	if err := publisher.Publish(ctx, pageURL); err != nil {
		t.Fatalf("publish: %v", err)
	}

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
}
