// Package jetstream asks the scrape service to read one page, by publishing a scrape request
// on the stream that service reads. A caller here waits only as long as one call may wait, so
// the request gives up rather than waiting for an origin that asks to be read later. The
// stream belongs to the service that reads it; until that stream exists, asking fails.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type ScrapeRequests struct {
	stream     jetstream.JetStream
	connection *nats.Conn
}

func OpenScrapeRequests(natsURL string) (*ScrapeRequests, error) {
	stream, connection, err := jetstreamconnect.Open(natsURL)
	if err != nil {
		return nil, err
	}
	return &ScrapeRequests{stream: stream, connection: connection}, nil
}

func (r *ScrapeRequests) AskToScrape(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) error {
	request, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL, GivesUpOnDeferral: true},
	)
	if err != nil {
		return fmt.Errorf("marshal the scrape request for %q: %w", pageURL, err)
	}
	if _, err := r.stream.Publish(
		ctx, pagescrapecontract.ScrapeRequestSubject, request,
	); err != nil {
		return fmt.Errorf("publish the scrape request for %q: %w", pageURL, err)
	}
	return nil
}

func (r *ScrapeRequests) Close() {
	r.connection.Close()
}
