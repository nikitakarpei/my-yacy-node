package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	crawlorderplacersjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/crawlorderplacers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

const (
	ordersSubject = "yacy.crawl.orders.test"
	canonicalURL  = "https://example.com/"
)

func ordersStream(t *testing.T) natsjetstream.JetStream {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateOrUpdateStream(context.Background(), natsjetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{ordersSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatalf("create orders stream: %v", err)
	}
	return js
}

func orderOnStream(t *testing.T, js natsjetstream.JetStream) yacycrawlcontract.CrawlOrder {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	consumer, err := js.CreateOrUpdateConsumer(ctx, yacycrawlcontract.OrdersStreamName,
		natsjetstream.ConsumerConfig{AckPolicy: natsjetstream.AckExplicitPolicy},
	)
	if err != nil {
		t.Fatalf("consume orders: %v", err)
	}
	message, err := consumer.Next()
	if err != nil {
		t.Fatalf("next order: %v", err)
	}
	order, err := yacycrawlcontract.UnmarshalCrawlOrder(message.Data())
	if err != nil {
		t.Fatalf("unmarshal order: %v", err)
	}
	return order
}

func TestPlacedOrderSeedsACrawlOfTheCanonicalURL(t *testing.T) {
	js := ordersStream(t)
	placement := crawlorderplacersjetstream.NewCrawlOrderPlacement(js, ordersSubject)

	if err := placement.Place(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, canonicalURL),
	); err != nil {
		t.Fatalf("place: %v", err)
	}

	order := orderOnStream(t, js)
	if len(order.SeedURLs) != 1 ||
		order.SeedURLs[0] != canonicalurltest.CanonicalURLOf(t, canonicalURL) {
		t.Errorf("seed urls = %v", order.SeedURLs)
	}
	if order.OrderID == "" {
		t.Error("order id empty")
	}
}

func TestPlacingAnOrderFailsWhenNoStreamCarriesTheSubject(t *testing.T) {
	js := ordersStream(t)
	placement := crawlorderplacersjetstream.NewCrawlOrderPlacement(js, "yacy.crawl.unbound")

	if err := placement.Place(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, canonicalURL),
	); err == nil {
		t.Fatal("expected an error when no stream carries the subject")
	}
}
