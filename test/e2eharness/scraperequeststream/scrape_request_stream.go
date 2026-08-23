//go:build e2e

// Package scraperequeststream provisions the scrape-request stream an e2e stack needs.
// No service creates it: the crawler and the shim publish to it, every corpus reads it,
// and an operator decides its retention. A suite stands in for that operator.
package scraperequeststream

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
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
		Name:      scraperequestcontract.ScrapeRequestsStreamName,
		Subjects:  []string{scraperequestcontract.ScrapeRequestSubject},
		Retention: jetstream.LimitsPolicy,
		MaxMsgs:   maxMsgs,
		Discard:   jetstream.DiscardOld,
	}); err != nil {
		t.Fatalf("provision the %s stream: %v", scraperequestcontract.ScrapeRequestsStreamName, err)
	}
}
