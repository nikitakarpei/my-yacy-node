//go:build e2e

// Package scraperequeststream provisions the scrape-request stream for a publish-only
// e2e stack that does not start the pagescrape service which owns the stream.
package scraperequeststream

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const maxMsgs = 1024

func Provision(t *testing.T, ctx context.Context, natsURL string) {
	t.Helper()
	conn, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats at %s: %v", natsURL, err)
	}
	defer conn.Close()
	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      pagescrapecontract.ScrapeRequestsStreamName,
		Subjects:  []string{pagescrapecontract.ScrapeRequestSubject},
		Retention: jetstream.LimitsPolicy,
		MaxMsgs:   maxMsgs,
		Discard:   jetstream.DiscardOld,
	}); err != nil {
		t.Fatalf("provision the %s stream: %v", pagescrapecontract.ScrapeRequestsStreamName, err)
	}
}
