// Package jetstream asks the crawl fleet to scrape one page, by publishing a scrape request
// on the stream the fleet reads. The stream belongs to the stack that reads it; until that
// stream exists, asking fails.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

type ScrapeRequests struct {
	stream               jetstream.JetStream
	connection           *nats.Conn
	scrapeRequestSubject string
}

func OpenScrapeRequests(natsURL, scrapeRequestSubject string) (*ScrapeRequests, error) {
	stream, connection, err := jetstreamconnect.Open(natsURL)
	if err != nil {
		return nil, err
	}
	return &ScrapeRequests{
		stream:               stream,
		connection:           connection,
		scrapeRequestSubject: scrapeRequestSubject,
	}, nil
}

func (r *ScrapeRequests) AskToScrape(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) error {
	request, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{PageURL: pageURL},
	)
	if err != nil {
		return fmt.Errorf("marshal the scrape request for %q: %w", pageURL, err)
	}
	if _, err := r.stream.Publish(ctx, r.scrapeRequestSubject, request); err != nil {
		return fmt.Errorf("publish the scrape request for %q: %w", pageURL, err)
	}
	return nil
}

func (r *ScrapeRequests) Close() {
	r.connection.Close()
}
