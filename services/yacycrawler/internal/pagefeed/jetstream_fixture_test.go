package pagefeed_test

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func startJetStream(t *testing.T) jetstream.JetStream {
	t.Helper()
	srv, err := natsserver.NewServer(&natsserver.Options{
		Port: -1, JetStream: true, StoreDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats not ready")
	}
	t.Cleanup(srv.Shutdown)
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	return js
}

func samplePage() crawlcapability.CrawledPage {
	return crawlcapability.CrawledPage{
		CanonicalURL: "http://example.com/a",
		Title:        "Hi",
		Language:     "en",
		CrawledAt:    time.Unix(1_700_000_000, 0),
	}
}

func consumeOne(t *testing.T, js jetstream.JetStream, stream string) []byte {
	t.Helper()
	consumer, err := js.CreateOrUpdateConsumer(context.Background(), stream,
		jetstream.ConsumerConfig{AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	_ = msg.Ack()
	return msg.Data()
}
