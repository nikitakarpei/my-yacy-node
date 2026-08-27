// Package jetstream publishes a scrape request for every page the crawler admits.
package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

type Publisher struct {
	stream jetstream.JetStream
}

func New(stream jetstream.JetStream) *Publisher {
	return &Publisher{stream: stream}
}

func (p *Publisher) Publish(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) error {
	data, err := scraperequestcontract.MarshalScrapeRequest(
		scraperequestcontract.ScrapeRequest{PageURL: pageURL},
	)
	if err != nil {
		return fmt.Errorf("marshal scrape request %s: %w", pageURL, err)
	}
	if _, err := p.stream.Publish(
		ctx,
		scraperequestcontract.ScrapeRequestSubject,
		data,
	); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", pageURL, err)
	}
	return nil
}
