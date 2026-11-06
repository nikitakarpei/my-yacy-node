package jetstream_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagerwistreams "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwistreams/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
)

func TestPageRWIsStreamKeepsAPageRWIOnlyWhileANodeHasToReadIt(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))

	stream, err := pagerwistreams.CreatePageRWIsStream(
		ctx, broker, pagerwistreams.PageRWIsStreamLimits{MaxBytes: 1 << 20},
	)
	if err != nil {
		t.Fatalf("create the rwis stream: %v", err)
	}
	if _, err := broker.Publish(ctx, pagerwicontract.PageRWISubject, []byte("{}")); err != nil {
		t.Fatalf("publish an rwi: %v", err)
	}
	info, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if info.State.Msgs != 0 {
		t.Errorf("the stream kept %d rwis no node reads, want none", info.State.Msgs)
	}
}

func TestPageRWIsStreamGivesEveryNodeConsumerEachPageRWI(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := pagerwistreams.CreatePageRWIsStream(
		ctx, broker, pagerwistreams.PageRWIsStreamLimits{MaxBytes: 1 << 20},
	)
	if err != nil {
		t.Fatalf("create the rwis stream: %v", err)
	}
	for _, durable := range []string{"first-nodes", "second-nodes"} {
		if _, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
			Durable:       durable,
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: pagerwicontract.PageRWISubject,
		}); err != nil {
			t.Fatalf("create the %s consumer: %v", durable, err)
		}
	}

	if _, err := broker.Publish(ctx, pagerwicontract.PageRWISubject, []byte("{}")); err != nil {
		t.Fatalf("publish an rwi: %v", err)
	}

	for _, durable := range []string{"first-nodes", "second-nodes"} {
		consumer, err := stream.Consumer(ctx, durable)
		if err != nil {
			t.Fatalf("open the %s consumer: %v", durable, err)
		}
		info, err := consumer.Info(ctx)
		if err != nil {
			t.Fatalf("%s consumer info: %v", durable, err)
		}
		if info.NumPending != 1 {
			t.Errorf(
				"%s has %d rwis to read, want the one published",
				durable,
				info.NumPending,
			)
		}
	}
}

func TestPageRWIsStreamRefusesAPageRWIWhileFull(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	pageRWI := make([]byte, 600)
	stream, err := pagerwistreams.CreatePageRWIsStream(
		ctx, broker, pagerwistreams.PageRWIsStreamLimits{MaxBytes: int64(len(pageRWI)) + 200},
	)
	if err != nil {
		t.Fatalf("create the rwis stream: %v", err)
	}
	if _, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "nodes",
		AckPolicy: jetstream.AckExplicitPolicy,
	}); err != nil {
		t.Fatalf("create a node consumer: %v", err)
	}
	if _, err := broker.Publish(ctx, pagerwicontract.PageRWISubject, pageRWI); err != nil {
		t.Fatalf("publish the first rwi: %v", err)
	}

	if _, err := broker.Publish(ctx, pagerwicontract.PageRWISubject, pageRWI); err == nil {
		t.Error("the full stream took another rwi")
	}
}
