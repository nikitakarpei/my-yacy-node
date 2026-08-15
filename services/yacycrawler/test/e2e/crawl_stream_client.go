//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	ordersSubject              = "yacy.crawl.orders"
	ordersStreamAppearanceWait = 60 * time.Second
	pageStreamAppearLimit      = 30 * time.Second
	pageStreamPollDelay        = 250 * time.Millisecond
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

func awaitOrdersStream(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	appeared := pollwait.For(ordersStreamAppearanceWait, func() bool {
		_, err := js.Stream(ctx, yacycrawlcontract.OrdersStreamName)
		return err == nil
	})
	if !appeared {
		t.Fatalf("orders stream did not appear within %s", ordersStreamAppearanceWait)
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

	var chunks []yacycrawlcontract.PageRWIChunk
	postingChunks := 0
	for {
		msg, err := consumer.Next(jetstream.FetchMaxWait(60 * time.Second))
		if err != nil {
			t.Fatalf("fetch page rwi chunk: %v", err)
		}
		chunk, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg.Data())
		if err != nil {
			t.Fatalf("decode page rwi chunk: %v", err)
		}
		chunks = append(chunks, chunk)
		if _, isPosting := chunk.(yacycrawlcontract.PageRWIPostingChunk); isPosting {
			postingChunks++
		}
		metadata, err := msg.Metadata()
		if err != nil {
			t.Fatalf("page rwi chunk metadata: %v", err)
		}
		if err := msg.Ack(); err != nil {
			t.Fatalf("ack: %v", err)
		}
		if metadata.NumPending == 0 && postingChunks > 0 {
			break
		}
	}
	representation, err := yacycrawlcontract.PageRWIRepresentationFromChunks(chunks)
	if err != nil {
		t.Fatalf("join page rwi chunks: %v", err)
	}
	return representation
}
