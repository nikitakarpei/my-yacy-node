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
	ordersSubject        = "yacy.crawl.orders"
	streamAppearanceWait = 60 * time.Second
	messageArrivalWait   = 60 * time.Second
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

func awaitStream(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	name string,
) jetstream.Stream {
	t.Helper()
	var stream jetstream.Stream
	appeared := pollwait.For(streamAppearanceWait, func() bool {
		found, err := js.Stream(ctx, name)
		if err != nil {
			return false
		}
		stream = found
		return true
	})
	if !appeared {
		t.Fatalf("crawler did not create the %s stream within %s", name, streamAppearanceWait)
	}
	return stream
}

func fetchOnePageRWIRepresentation(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
) yacycrawlcontract.PageRWIRepresentation {
	t.Helper()
	stream := awaitStream(t, ctx, js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindRWI))
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create crawled page rwi consumer: %v", err)
	}

	var chunks []yacycrawlcontract.PageRWIChunk
	postingChunks := 0
	for {
		msg, err := consumer.Next(jetstream.FetchMaxWait(messageArrivalWait))
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

func fetchOneReachedPage(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
) yacycrawlcontract.ReachedPage {
	t.Helper()
	stream := awaitStream(t, ctx, js, yacycrawlcontract.ReachedPagesStreamName)
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create reached page consumer: %v", err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(messageArrivalWait))
	if err != nil {
		t.Fatalf("fetch reached page: %v", err)
	}
	page, err := yacycrawlcontract.UnmarshalReachedPage(msg.Data())
	if err != nil {
		t.Fatalf("decode reached page: %v", err)
	}
	if err := msg.Ack(); err != nil {
		t.Fatalf("ack reached page: %v", err)
	}
	return page
}
