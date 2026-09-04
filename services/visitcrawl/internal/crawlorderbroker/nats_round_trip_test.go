package crawlorderbroker_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const crawlOrdersSubject = "yacy.crawl.orders"

type recordingCrawlOrderPublicationObserver struct {
	publishedOrderID string
	publishedSubject string
	failures         int
}

func (o *recordingCrawlOrderPublicationObserver) CrawlOrderPublished(
	_ context.Context,
	orderID string,
	subject string,
) {
	o.publishedOrderID = orderID
	o.publishedSubject = subject
}

func (o *recordingCrawlOrderPublicationObserver) CrawlOrderPublishingFailed(
	_ context.Context,
	_ string,
	_ string,
	_ error,
) {
	o.failures++
}

func (o *recordingCrawlOrderPublicationObserver) CrawlOrderEncodingFailed(
	_ context.Context,
	_ string,
	_ error,
) {
	o.failures++
}

func createOrdersStream(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	if _, err := natstestserver.ConnectJetStream(t, url).CreateOrUpdateStream(
		ctx,
		jetstream.StreamConfig{
			Name:      yacycrawlcontract.OrdersStreamName,
			Subjects:  []string{crawlOrdersSubject},
			Retention: jetstream.WorkQueuePolicy,
		},
	); err != nil {
		t.Fatalf("create orders stream: %v", err)
	}
}

func TestPublishedCrawlOrderReachesOrdersStream(t *testing.T) {
	url := natstestserver.Start(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	createOrdersStream(t, ctx, url)

	observer := &recordingCrawlOrderPublicationObserver{}
	broker, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:            url,
		CrawlOrdersSubject: crawlOrdersSubject,
	}, observer)
	if err != nil {
		t.Fatalf("open broker: %v", err)
	}
	t.Cleanup(broker.Close)

	order := yacycrawlcontract.CrawlOrder{
		OrderID: "order-1",
		Profile: yacycrawlcontract.CrawlProfile{MaxDepth: 2},
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "https://example.org"),
		},
	}
	broker.OrderPlacer.Place(ctx, order)

	js := natstestserver.ConnectJetStream(t, url)
	consumer, err := js.CreateOrUpdateConsumer(
		ctx,
		yacycrawlcontract.OrdersStreamName,
		jetstream.ConsumerConfig{
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: crawlOrdersSubject,
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
	if got.OrderID != order.OrderID || got.Profile.MaxDepth != order.Profile.MaxDepth {
		t.Fatalf("round-tripped order mismatch: %+v", got)
	}
	if observer.publishedOrderID != order.OrderID ||
		observer.publishedSubject != crawlOrdersSubject {
		t.Fatalf("published %q on %q", observer.publishedOrderID, observer.publishedSubject)
	}
	if observer.failures != 0 {
		t.Fatalf("failures = %d, want 0", observer.failures)
	}
}

func TestOpenRejectsUnreachableNATS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:            "nats://127.0.0.1:1",
		CrawlOrdersSubject: crawlOrdersSubject,
	}, &recordingCrawlOrderPublicationObserver{}); err == nil {
		t.Fatal("unreachable nats should fail to open")
	}
}
