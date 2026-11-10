package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

// TECHDEBT: Failure rule — OfferPage and ReportScrapeFailure return the publish error to their consumer instead of reporting it to an observer of their own.
type PageOfferPublisher struct {
	pageOffers jetstream.JetStream
}

func NewPageOfferPublisher(pageOffers jetstream.JetStream) *PageOfferPublisher {
	return &PageOfferPublisher{pageOffers: pageOffers}
}

func (p *PageOfferPublisher) OfferPage(
	ctx context.Context,
	offer pagescrapecontract.PageOfferMessage,
) error {
	if _, err := p.pageOffers.PublishMsg(ctx, natsMessageOf(offer)); err != nil {
		return fmt.Errorf("offer a page: %w", err)
	}
	return nil
}

func natsMessageOf(offer pagescrapecontract.PageOfferMessage) *nats.Msg {
	message := nats.NewMsg(pagescrapecontract.OfferedPageSubject)
	message.Data = offer.Body
	if attestation, attested := offer.Attestation.Get(); attested {
		message.Header.Set(pagescrapecontract.PageOfferAttestationHeader, string(attestation))
	}
	return message
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
