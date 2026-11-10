// Package pageintake builds the reverse word index of each page the scrape service offers,
// publishes it for the nodes, and sends a receipt back for the caller waiting on that page.
// It takes the next page only while the nodes keep up with the RWIs published.
package pageintake

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

type PageRWIs interface {
	Publish(ctx context.Context, pageRWI pagerwicontract.PageRWI) bool
}

type IntakeReceipts interface {
	ReportKeptPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ReportRejectedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type OfferedPageConsumer struct {
	offeredPageSource          pullintake.MessageSource
	formatDerivations          pageformats.FormatDerivationCatalog
	pageIndexBuilder           pagerwi.IndexBuilder
	pageRWIs                   PageRWIs
	pageAttestations           PageAttestations
	admission                  pullintake.Admission
	intakeReceipts             IntakeReceipts
	pageIntakeObserver         PageIntakeObserver
	pageOfferIntakeConcurrency int
}

type OfferedPageConsumerConfig struct {
	OfferedPageSource          pullintake.MessageSource
	FormatDerivations          pageformats.FormatDerivationCatalog
	PageIndexBuilder           pagerwi.IndexBuilder
	PageRWIs                   PageRWIs
	PageAttestations           PageAttestations
	Admission                  pullintake.Admission
	IntakeReceipts             IntakeReceipts
	PageIntakeObserver         PageIntakeObserver
	PageOfferIntakeConcurrency int
}

func NewOfferedPageConsumer(config OfferedPageConsumerConfig) *OfferedPageConsumer {
	return &OfferedPageConsumer{
		offeredPageSource:          config.OfferedPageSource,
		formatDerivations:          config.FormatDerivations,
		pageIndexBuilder:           config.PageIndexBuilder,
		pageRWIs:                   config.PageRWIs,
		pageAttestations:           config.PageAttestations,
		admission:                  config.Admission,
		intakeReceipts:             config.IntakeReceipts,
		pageIntakeObserver:         config.PageIntakeObserver,
		pageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
	}
}

func (c *OfferedPageConsumer) Run(ctx context.Context) error {
	return pullintake.RunAdmittedBy(
		ctx,
		pullintake.MessageQueueOf(c.offeredPageSource),
		c.pageOfferIntakeConcurrency,
		c.admission,
		c.takeIn,
	)
}

func (c *OfferedPageConsumer) takeIn(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	page, err := pagescrapecontract.UnmarshalOfferedPage(message.Body())
	if err != nil {
		c.pageIntakeObserver.OfferedPageInvalid(ctx)

		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	c.pageIntakeObserver.PageOffered(ctx, message.Identity(), page.PageURL)
	document, extracted := c.documentOf(ctx, message, page)
	if !extracted {
		return c.reject(ctx, message, page.PageURL)
	}
	text, derived := c.fullTextOf(ctx, document, page.LandedURL)
	if !derived {
		c.pageIntakeObserver.NoIndexDerived(ctx, message.Identity(), page.PageURL)

		return c.reject(ctx, message, page.PageURL)
	}
	c.publishAttested(ctx, message, page, c.pageIndexBuilder.RWIFrom(ctx, page, document, text))

	return nil
}

func (c *OfferedPageConsumer) documentOf(
	ctx context.Context,
	message pullintake.PendingMessage,
	page pagescrapecontract.OfferedPage,
) (documentextraction.Document, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, page.Body, page.ContentType, page.LandedURL,
	)
	if errors.Is(err, documentextraction.ErrUnsupportedMediaType) {
		c.pageIntakeObserver.MediaTypeUnsupported(
			ctx,
			message.Identity(),
			page.LandedURL,
			page.ContentType,
		)

		return documentextraction.Document{}, false
	}
	if err != nil {
		c.pageIntakeObserver.DocumentExtractionFailed(
			ctx,
			message.Identity(),
			page.LandedURL,
			err,
		)

		return documentextraction.Document{}, false
	}

	return document, true
}

func (c *OfferedPageConsumer) fullTextOf(
	ctx context.Context,
	document documentextraction.Document,
	landedURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	return c.formatDerivations.BytesIn(
		ctx, documentextraction.FormatFullText, document, landedURL,
	)
}

func (c *OfferedPageConsumer) publishAttested(
	ctx context.Context,
	message pullintake.PendingMessage,
	page pagescrapecontract.OfferedPage,
	pageRWI pagerwicontract.PageRWI,
) {
	attestation, ready := c.pageAttestations.AttestationOf(ctx, page, pageRWI)
	if !ready {
		message.Return(ctx)

		return
	}
	c.publish(ctx, message, page.PageURL, attestedBy(pageRWI, attestation))
}

func attestedBy(
	pageRWI pagerwicontract.PageRWI,
	attestation json.RawMessage,
) pagerwicontract.PageRWI {
	pageRWI.Attestation = attestation

	return pageRWI
}

func (c *OfferedPageConsumer) publish(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	pageRWI pagerwicontract.PageRWI,
) {
	if !c.pageRWIs.Publish(ctx, pageRWI) {
		c.pageIntakeObserver.PageRWINotPublished(ctx, message.Identity(), pageURL)
		message.Return(ctx)

		return
	}
	c.intakeReceipts.ReportKeptPage(ctx, pageURL)
	message.Acknowledge(ctx)
	c.pageIntakeObserver.PageRWIBuilt(ctx, message.Identity(), pageURL, len(pageRWI.Postings))
}

func (c *OfferedPageConsumer) reject(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
) error {
	c.intakeReceipts.ReportRejectedPage(ctx, pageURL)
	message.Acknowledge(ctx)

	return nil
}
