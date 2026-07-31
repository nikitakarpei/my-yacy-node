package crawlorderbroker_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/crawlorderbroker"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const ordersSubject = "yacy.crawl.orders"

func createOrdersStream(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	if err := yacycrawlcontract.EnsureOrdersStream(
		ctx,
		natstestserver.ConnectJetStream(t, url),
		yacycrawlcontract.OrdersStreamSpec{Subject: ordersSubject},
	); err != nil {
		t.Fatalf("create orders stream: %v", err)
	}
}

func TestOrderPlacementDeliversToOrdersStream(t *testing.T) {
	url := natstestserver.Start(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	createOrdersStream(t, ctx, url)

	broker, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:       url,
		OrdersSubject: ordersSubject,
	})
	if err != nil {
		t.Fatalf("open broker: %v", err)
	}
	t.Cleanup(broker.Close)

	order := yacycrawlcontract.CrawlOrder{
		OrderID:  "order-1",
		Profile:  yacycrawlcontract.CrawlProfile{Name: "docs"},
		SeedURLs: []string{"https://example.org"},
	}
	if err := broker.Orders.Place(ctx, order); err != nil {
		t.Fatalf("place order: %v", err)
	}

	js := natstestserver.ConnectJetStream(t, url)
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
	if got.OrderID != order.OrderID || got.Profile.Name != order.Profile.Name {
		t.Fatalf("round-tripped order mismatch: %+v", got)
	}
}

func TestOpenRejectsUnreachableNATS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := crawlorderbroker.Open(ctx, crawlorderbroker.Config{
		NATSURL:       "nats://127.0.0.1:1",
		OrdersSubject: ordersSubject,
	}); err == nil {
		t.Fatal("unreachable nats should fail to open")
	}
}
