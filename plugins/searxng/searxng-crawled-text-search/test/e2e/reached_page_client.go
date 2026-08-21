//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const reachedPageMax = 64

func connectJetStream(t *testing.T, natsURL string) jetstream.JetStream {
	t.Helper()
	nc, err := nats.Connect(natsURL)
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

func createReachedPagesStream(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.ReachedPagesStreamName,
		Subjects:  []string{yacycrawlcontract.ReachedPageSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   reachedPageMax,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		t.Fatalf("create reached pages stream: %v", err)
	}
}

func publishReachedCorpus(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	for _, pageURL := range reachedPageURLs() {
		publishReachedPage(t, ctx, js, pageURL)
	}
}

func publishReachedPage(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	canonicalURL string,
) {
	t.Helper()
	data, err := yacycrawlcontract.MarshalReachedPage(
		yacycrawlcontract.ReachedPage{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, canonicalURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal reached page: %v", err)
	}
	if _, err := js.Publish(ctx, yacycrawlcontract.ReachedPageSubject, data); err != nil {
		t.Fatalf("publish reached page: %v", err)
	}
}
