package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeRequests interface {
	Publish(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) error
}

type ScrapeRequestObserver interface {
	ScrapeRequestPublished(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ScrapeRequestPublicationFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
}

type ScrapeRequestObservers []ScrapeRequestObserver

func (observers ScrapeRequestObservers) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ScrapeRequestPublished(ctx, pageURL)
	}
}

func (observers ScrapeRequestObservers) ScrapeRequestPublicationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeRequestPublicationFailed(ctx, pageURL, cause)
	}
}

type ScrapeRequestPublisher struct {
	inner    ScrapeRequests
	observer ScrapeRequestObserver
}

func NewScrapeRequestPublisher(
	inner ScrapeRequests,
	observer ScrapeRequestObserver,
) *ScrapeRequestPublisher {
	return &ScrapeRequestPublisher{inner: inner, observer: observer}
}

func (p *ScrapeRequestPublisher) Publish(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) error {
	if err := p.inner.Publish(ctx, pageURL); err != nil {
		p.observer.ScrapeRequestPublicationFailed(ctx, pageURL, err)
		return err
	}
	p.observer.ScrapeRequestPublished(ctx, pageURL)
	return nil
}
