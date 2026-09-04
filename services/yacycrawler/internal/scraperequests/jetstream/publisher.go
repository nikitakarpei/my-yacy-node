// Package jetstream publishes a scrape request for every page the crawler admits. A request
// that never leaves goes to the observer; the caller learns nothing and visits the next page.
package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type Publisher struct {
	stream   jetstream.JetStream
	observer ScrapeRequestPublicationObserver
}

func New(
	stream jetstream.JetStream,
	observer ScrapeRequestPublicationObserver,
) *Publisher {
	return &Publisher{stream: stream, observer: observer}
}

func (p *Publisher) Publish(ctx context.Context, pageURL canonicalurl.CanonicalURL) {
	data, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL},
	)
	if err != nil {
		p.observer.ScrapeRequestEncodingFailed(ctx, pageURL, err)
		return
	}
	if _, err := p.stream.Publish(
		ctx,
		pagescrapecontract.ScrapeRequestSubject,
		data,
	); err != nil {
		p.observer.ScrapeRequestPublishingFailed(ctx, pageURL, err)
		return
	}
	p.observer.ScrapeRequestPublished(ctx, pageURL)
}
