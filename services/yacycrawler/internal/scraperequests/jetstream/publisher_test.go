package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/scraperequests/jetstream"
)

func TestPublishWritesContractMessage(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, natsjetstream.StreamConfig{
		Name:      scraperequestcontract.ScrapeRequestsStreamName,
		Subjects:  []string{scraperequestcontract.ScrapeRequestSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	publisher := jetstream.New(js)

	const url = "http://example.com/a"
	canonicalURL := canonicalurltest.CanonicalURLOf(t, url)
	if err := publisher.Publish(ctx, canonicalURL); err != nil {
		t.Fatalf("publish: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, scraperequestcontract.ScrapeRequestsStreamName,
		natsjetstream.ConsumerConfig{AckPolicy: natsjetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(natsjetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	_ = msg.Ack()

	page, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if page.CanonicalURL != canonicalURL {
		t.Fatalf("canonical url = %q, want %q", page.CanonicalURL, url)
	}
}
