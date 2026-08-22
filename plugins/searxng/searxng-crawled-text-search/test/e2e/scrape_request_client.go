//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
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

func publishScrapedCorpus(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	for _, pageURL := range scrapedCorpusURLs() {
		publishScrapeRequest(t, ctx, js, pageURL)
	}
}

func publishScrapeRequest(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	canonicalURL string,
) {
	t.Helper()
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, canonicalURL),
		},
	)
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	if _, err := js.Publish(ctx, scraperequestcontract.ScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
	}
}
