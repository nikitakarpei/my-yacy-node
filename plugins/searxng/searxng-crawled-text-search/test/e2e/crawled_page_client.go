//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const crawledPageMax = 64

var crawledPageSubject = yacycrawlcontract.CrawledPageSubject(
	yacycrawlcontract.PageRepresentationKindText,
)

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

func createCrawledPageStream(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: yacycrawlcontract.CrawledPageStreamName(
			yacycrawlcontract.PageRepresentationKindText,
		),
		Subjects:  []string{crawledPageSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   crawledPageMax,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		t.Fatalf("create crawled page stream: %v", err)
	}
}

func publishCrawledCorpus(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	for _, page := range crawledPages() {
		publishCrawledPage(t, ctx, js, page)
	}
}

func publishCrawledPage(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	page yacycrawlcontract.PageTextRepresentation,
) {
	t.Helper()
	data, err := yacycrawlcontract.MarshalPageTextRepresentation(page)
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	if _, err := js.Publish(ctx, crawledPageSubject, data); err != nil {
		t.Fatalf("publish crawled page: %v", err)
	}
}
