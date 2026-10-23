// Package jetstream asks the scrape service to read one page, by publishing a scrape request
// on the stream that service owns. How each request fared goes to the observer; the caller
// learns only whether the request left.
package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type ScrapeRequestPublicationObserver interface {
	ScrapeRequestPublished(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ScrapeRequestNotPublished(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
}

type ScrapeRequests struct {
	broker   jetstream.JetStream
	observer ScrapeRequestPublicationObserver
}

func NewScrapeRequests(
	broker jetstream.JetStream,
	observer ScrapeRequestPublicationObserver,
) *ScrapeRequests {
	return &ScrapeRequests{broker: broker, observer: observer}
}

func (r *ScrapeRequests) AskToScrape(ctx context.Context, pageURL canonicalurl.CanonicalURL) bool {
	request, err := pagescrapecontract.MarshalScrapeRequest(
		pagescrapecontract.ScrapeRequest{PageURL: pageURL},
	)
	if err != nil {
		r.observer.ScrapeRequestNotPublished(ctx, pageURL, err)
		return false
	}
	if _, err := r.broker.Publish(
		ctx,
		pagescrapecontract.ScrapeRequestSubject,
		request,
	); err != nil {
		r.observer.ScrapeRequestNotPublished(ctx, pageURL, err)
		return false
	}
	r.observer.ScrapeRequestPublished(ctx, pageURL)
	return true
}

type ScrapeRequestPublicationObservers []ScrapeRequestPublicationObserver

func (observers ScrapeRequestPublicationObservers) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ScrapeRequestPublished(ctx, pageURL)
	}
}

func (observers ScrapeRequestPublicationObservers) ScrapeRequestNotPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeRequestNotPublished(ctx, pageURL, cause)
	}
}
