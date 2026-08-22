package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/orderreceivers/jetstream"
)

const ordersSubject = "yacy.crawl.orders"

func startJetStream(t *testing.T) natsjetstream.JetStream {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateOrUpdateStream(context.Background(), natsjetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{ordersSubject},
		Retention: natsjetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	return js
}

func consumer(t *testing.T, js natsjetstream.JetStream) natsjetstream.Consumer {
	t.Helper()
	c, err := js.CreateOrUpdateConsumer(
		context.Background(),
		yacycrawlcontract.OrdersStreamName,
		natsjetstream.ConsumerConfig{
			Durable:   "yacycrawler",
			AckPolicy: natsjetstream.AckExplicitPolicy,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReceiverDeliversDecodedOrder(t *testing.T) {
	js := startJetStream(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	payload, err := yacycrawlcontract.MarshalCrawlOrder(yacycrawlcontract.CrawlOrder{
		OrderID: "o1",
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://a.com/"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(ctx, ordersSubject, payload); err != nil {
		t.Fatal(err)
	}

	receiver, err := jetstream.NewOrderReceiver(ctx, consumer(t, js))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case delivery := <-receiver.Deliveries():
		if delivery.Order().OrderID != "o1" {
			t.Fatalf("order id = %q", delivery.Order().OrderID)
		}
		if err := delivery.ExtendOwnership(ctx); err != nil {
			t.Fatalf("extend ownership: %v", err)
		}
		if err := delivery.Acknowledge(ctx); err != nil {
			t.Fatalf("ack: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no delivery received")
	}
}

func TestReceiverTermsUndecodableOrder(t *testing.T) {
	js := startJetStream(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := js.Publish(ctx, ordersSubject, []byte("not json")); err != nil {
		t.Fatal(err)
	}
	receiver, err := jetstream.NewOrderReceiver(ctx, consumer(t, js))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-receiver.Deliveries():
		t.Fatal("undecodable order should not be delivered")
	case <-time.After(500 * time.Millisecond):
	}
}
