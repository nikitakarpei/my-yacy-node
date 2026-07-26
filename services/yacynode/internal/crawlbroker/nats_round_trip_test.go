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

const ingestSubject = "yacy.crawl.ingest"

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
	broker, err := crawlbroker.Open(ctx, crawlbroker.Config{
		NATSURL:       url,
		IngestSubject: ingestSubject,
		IngestDurable: "yacy-node",
	})
	if err != nil {
		t.Fatalf("open broker: %v", err)
	}
	t.Cleanup(broker.Close)
	return broker, js, ctx
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
