package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type PageOfferPublisher struct {
	pageOffers jetstream.JetStream
}

func NewPageOfferPublisher(pageOffers jetstream.JetStream) *PageOfferPublisher {
	return &PageOfferPublisher{pageOffers: pageOffers}
}

func (p *PageOfferPublisher) OfferPage(
	ctx context.Context,
	page pagescrapecontract.OfferedPage,
) error {
	data, err := pagescrapecontract.MarshalOfferedPage(page)
	if err != nil {
		return err
	}
	if _, err := p.pageOffers.Publish(
		ctx,
		pagescrapecontract.OfferedPageSubject,
		data,
	); err != nil {
		return fmt.Errorf("offer the page %q: %w", page.PageURL, err)
	}
	return nil
}

func (p *PageOfferPublisher) ReportScrapeFailure(
	ctx context.Context,
	failure pagescrapecontract.ScrapeFailure,
) error {
	data, err := pagescrapecontract.MarshalScrapeFailure(failure)
	if err != nil {
		return err
	}
	if _, err := p.pageOffers.Publish(
		ctx, pagescrapecontract.ScrapeFailureSubject, data,
	); err != nil {
		return fmt.Errorf("report the failed scrape of %q: %w", failure.PageURL, err)
	}
	return nil
}
