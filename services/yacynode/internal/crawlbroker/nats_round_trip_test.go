package crawlbroker_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/crawlbroker"
)

const (
	ordersSubject = "yacy.crawl.orders"
	ingestSubject = "yacy.crawl.ingest"
)

func openBroker(t *testing.T) (*crawlbroker.CrawlBroker, jetstream.JetStream, context.Context) {
	t.Helper()
	url := natstestserver.Start(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	js := natstestserver.ConnectJetStream(t, url)
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx, js, yacycrawlcontract.PageRepresentationKindRWI,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: ingestSubject, MaxMsgs: 16},
	); err != nil {
		t.Fatalf("create ingest stream: %v", err)
	}
	if err := yacycrawlcontract.EnsureOrdersStream(
		ctx, js, yacycrawlcontract.OrdersStreamSpec{Subject: ordersSubject},
	); err != nil {
		t.Fatalf("create orders stream: %v", err)
	}
	broker, err := crawlbroker.Open(ctx, crawlbroker.Config{
		NATSURL:       url,
		OrdersSubject: ordersSubject,
		IngestSubject: ingestSubject,
		IngestDurable: "yacy-node",
	})
	if err != nil {
		t.Fatalf("open broker: %v", err)
	}
	t.Cleanup(broker.Close)
	return broker, js, ctx
}

func TestOrderPublisherDeliversToOrdersStream(t *testing.T) {
	broker, js, ctx := openBroker(t)

	order := yacycrawlcontract.CrawlOrder{
		OrderID:  "order-1",
		Profile:  yacycrawlcontract.NewCrawlProfile(yacycrawlcontract.CrawlProfile{Name: "docs"}),
		SeedURLs: []string{"https://example.org"},
	}
	if err := broker.Orders.Publish(ctx, order); err != nil {
		t.Fatalf("publish order: %v", err)
	}

	consumer, err := js.CreateOrUpdateConsumer(
		ctx,
		yacycrawlcontract.OrdersStreamName,
		jetstream.ConsumerConfig{
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: ordersSubject,
		},
	)
	if err != nil {
		t.Fatalf("orders consumer: %v", err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("fetch order: %v", err)
	}
	got, err := yacycrawlcontract.UnmarshalCrawlOrder(msg.Data())
	if err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if got.OrderID != order.OrderID || got.Profile.Handle != order.Profile.Handle {
		t.Fatalf("round-tripped order mismatch: %+v", got)
	}
}

func TestIngestReceiverDeliversDecodableBatchAndSkipsGarbage(t *testing.T) {
	broker, js, ctx := openBroker(t)

	if _, err := js.Publish(ctx, ingestSubject, []byte("not json")); err != nil {
		t.Fatalf("publish garbage: %v", err)
	}
	chunk := yacycrawlcontract.PageRWIMetadataChunk{
		CanonicalURL: "https://example.org",
	}
	data, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal chunk: %v", err)
	}
	if _, err := js.Publish(ctx, ingestSubject, data); err != nil {
		t.Fatalf("publish chunk: %v", err)
	}

	select {
	case delivery := <-broker.Ingest.Receive():
		delivered, ok := delivery.Chunk.(yacycrawlcontract.PageRWIMetadataChunk)
		if !ok {
			t.Fatalf("chunk = %T, want PageRWIMetadataChunk", delivery.Chunk)
		}
		if delivered.CanonicalURL != chunk.CanonicalURL {
			t.Fatalf(
				"canonicalURL = %q, want %q",
				delivered.CanonicalURL,
				chunk.CanonicalURL,
			)
		}
		if err := delivery.Ack(ctx); err != nil {
			t.Fatalf("ack: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no ingest delivery received")
	}
}
