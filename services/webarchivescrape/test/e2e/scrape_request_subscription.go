//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type scrapeRequestSubscription struct {
	connection   *nats.Conn
	subscription *nats.Subscription
}

func subscribeToScrapeRequests(t *testing.T, natsURL string) scrapeRequestSubscription {
	t.Helper()
	connection, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect scrape-request subscription: %v", err)
	}
	t.Cleanup(connection.Close)
	subscription, err := connection.SubscribeSync(pagescrapecontract.ScrapeRequestSubject)
	if err != nil {
		t.Fatalf("subscribe to scrape requests: %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("flush scrape-request subscription: %v", err)
	}
	return scrapeRequestSubscription{connection: connection, subscription: subscription}
}

func (s scrapeRequestSubscription) next(t *testing.T) pagescrapecontract.ScrapeRequest {
	t.Helper()
	message, err := s.subscription.NextMsg(10 * time.Second)
	if err != nil {
		t.Fatalf("receive scrape request: %v", err)
	}
	request, err := pagescrapecontract.UnmarshalScrapeRequest(message.Data)
	if err != nil {
		t.Fatalf("read scrape request: %v", err)
	}
	return request
}

func (s scrapeRequestSubscription) hasMessage() bool {
	_, err := s.subscription.NextMsg(500 * time.Millisecond)
	return err == nil
}
