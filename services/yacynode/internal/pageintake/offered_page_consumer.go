// Package pageintake stores the reverse word index of each page the scrape service offers,
// and sends a receipt back for the caller waiting on that page.
package pageintake

import (
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
)

type IntakeReceipts interface {
	ReportKeptPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ReportRejectedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type OfferedPageConsumer struct {
	offeredPageSource          pullintake.MessageSource
	formatDerivations          pageformats.FormatDerivationCatalog
	pageIndexBuilder           pagerwi.IndexBuilder
	pageReceiver               pageadmission.PageReceiver
	intakeReceipts             IntakeReceipts
	pageIntakeObserver         PageIntakeObserver
	pageOfferIntakeConcurrency int
}

type OfferedPageConsumerConfig struct {
	OfferedPageSource          pullintake.MessageSource
	FormatDerivations          pageformats.FormatDerivationCatalog
	PageIndexBuilder           pagerwi.IndexBuilder
	PageReceiver               pageadmission.PageReceiver
	IntakeReceipts             IntakeReceipts
	PageIntakeObserver         PageIntakeObserver
	PageOfferIntakeConcurrency int
}

func NewOfferedPageConsumer(config OfferedPageConsumerConfig) *OfferedPageConsumer {
	return &OfferedPageConsumer{
		offeredPageSource:          config.OfferedPageSource,
		formatDerivations:          config.FormatDerivations,
		pageIndexBuilder:           config.PageIndexBuilder,
		pageReceiver:               config.PageReceiver,
		intakeReceipts:             config.IntakeReceipts,
		pageIntakeObserver:         config.PageIntakeObserver,
		pageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
	}
}

func (c *OfferedPageConsumer) Run(ctx context.Context) error {
	return pullintake.Run(
		ctx,
		pullintake.MessageQueueOf(c.offeredPageSource),
		c.pageOfferIntakeConcurrency,
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
	reachedAt := time.Now()
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
	c.store(
		ctx,
		message,
		page.PageURL,
		c.pageIndexBuilder.RWIFrom(ctx, page, document, text, reachedAt),
	)

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

func (c *OfferedPageConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	pageRWI yacymodel.Optional[pagerwi.PageRWI],
) {
	if !c.admitPage(ctx, message, pageURL, pageRWI) {
		return
	}
	c.intakeReceipts.ReportKeptPage(ctx, pageURL)
	message.Acknowledge(ctx)
	c.pageIntakeObserver.PageIndexed(ctx, message.Identity(), pageURL)
}

func (c *OfferedPageConsumer) admitPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	pageRWI yacymodel.Optional[pagerwi.PageRWI],
) bool {
	amountOfPostings := amountOfPostingsOf(pageRWI)
	receipt, err := c.pageReceiver.Receive(ctx, yacymodel.URLHashOf(pageURL.WebAddress()), pageRWI)
	if err != nil {
		c.pageIntakeObserver.PageAdmissionFailed(
			ctx,
			message.Identity(),
			pageURL,
			amountOfPostings,
			err,
		)
		message.Return(ctx)

		return false
	}
	if receipt.Busy {
		c.pageIntakeObserver.PageAdmissionBusy(
			ctx,
			message.Identity(),
			pageURL,
			amountOfPostings,
		)
		message.Return(ctx)

		return false
	}
	c.pageIntakeObserver.PageAdmitted(
		ctx,
		message.Identity(),
		pageURL,
		amountOfPostings,
	)

	return true
}

func amountOfPostingsOf(pageRWI yacymodel.Optional[pagerwi.PageRWI]) int {
	rwi, indexed := pageRWI.Get()
	if !indexed {
		return 0
	}

	return rwi.AmountOfPostings()
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
