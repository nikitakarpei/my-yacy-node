//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	ordersSubject         = "yacy.crawl.orders"
	pageStreamAppearLimit = 30 * time.Second
	pageStreamPollDelay   = 250 * time.Millisecond
)

func connectJetStream(t *testing.T, url string) jetstream.JetStream {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func ensureOrdersStream(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	if err := yacycrawlcontract.EnsureOrdersStream(ctx, js, yacycrawlcontract.OrdersStreamSpec{
		Subject: ordersSubject,
	}); err != nil {
		t.Fatalf("ensure orders stream: %v", err)
	}
}

func awaitPageStream(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	representation yacycrawlcontract.PageRepresentationKind,
) jetstream.Stream {
	t.Helper()
	name := yacycrawlcontract.CrawledPageStreamName(representation)
	deadline := time.Now().Add(pageStreamAppearLimit)
	for {
		stream, err := js.Stream(ctx, name)
		if err == nil {
			return stream
		}
		if time.Now().After(deadline) {
			t.Fatalf("crawler did not create the %s stream: %v", name, err)
		}
		time.Sleep(pageStreamPollDelay)
	}
}

func fetchOnePageRWIRepresentation(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
) yacycrawlcontract.PageRWIRepresentation {
	t.Helper()
	stream := awaitPageStream(t, ctx, js, yacycrawlcontract.PageRepresentationKindRWI)
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create crawled page rwi consumer: %v", err)
	}

	var representation yacycrawlcontract.PageRWIRepresentation
	for len(representation.Postings) == 0 {
		msg, err := consumer.Next(jetstream.FetchMaxWait(60 * time.Second))
		if err != nil {
			t.Fatalf("fetch page rwi chunk: %v", err)
		}
		chunk, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg.Data())
		if err != nil {
			t.Fatalf("decode page rwi chunk: %v", err)
		}
		switch chunk := chunk.(type) {
		case yacycrawlcontract.PageRWIMetadataChunk:
			if representation.CanonicalURL == "" {
				representation.CanonicalURL = chunk.CanonicalURL
			}
			representation.Metadata = append(representation.Metadata, chunk.Metadata...)
		case yacycrawlcontract.PageRWIPostingChunk:
			if representation.CanonicalURL == "" {
				representation.CanonicalURL = chunk.CanonicalURL
			}
			representation.Postings = append(representation.Postings, chunk.Postings...)
		}
		if err := msg.Ack(); err != nil {
			t.Fatalf("ack: %v", err)
		}
	}
	return representation
}
