// Package pageintake stores the reverse word index of each page the scrape service offers:
// the page's URL metadata, then its postings, and sends a receipt back for the caller waiting
// on that page.
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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

var errURLMetadataAdmissionRejected = errors.New("url metadata admission rejected")

type IntakeReceipts interface {
	ReportKeptPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ReportRejectedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type OfferedPageConsumer struct {
	offeredPageSource          pullintake.MessageSource
	formatDerivations          pageformats.FormatDerivationCatalog
	urlReceiver                urlmeta.URLReceiver
	postingReceiver            rwiadmission.PostingReceiver
	intakeReceipts             IntakeReceipts
	pageIntakeObserver         PageIntakeObserver
	pageOfferIntakeConcurrency int
}

type OfferedPageConsumerConfig struct {
	OfferedPageSource          pullintake.MessageSource
	FormatDerivations          pageformats.FormatDerivationCatalog
	URLReceiver                urlmeta.URLReceiver
	PostingReceiver            rwiadmission.PostingReceiver
	IntakeReceipts             IntakeReceipts
	PageIntakeObserver         PageIntakeObserver
	PageOfferIntakeConcurrency int
}

func NewOfferedPageConsumer(config OfferedPageConsumerConfig) *OfferedPageConsumer {
	return &OfferedPageConsumer{
		offeredPageSource:          config.OfferedPageSource,
		formatDerivations:          config.FormatDerivations,
		urlReceiver:                config.URLReceiver,
		postingReceiver:            config.PostingReceiver,
		intakeReceipts:             config.IntakeReceipts,
		pageIntakeObserver:         config.PageIntakeObserver,
		pageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
	}
}

func (c *OfferedPageConsumer) Run(ctx context.Context) error {
	return pullintake.Run(
		ctx,
		c.offeredPageSource,
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
	c.store(ctx, message, pagerwi.Of(page, document, text, reachedAt))

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
	return c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, document, landedURL,
	)
}

func (c *OfferedPageConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	index pagerwi.PageRWI,
) {
	if !c.admitURLMetadata(ctx, message, index) {
		return
	}
	if !c.admitPostings(ctx, message, index) {
		return
	}
	c.intakeReceipts.ReportKeptPage(ctx, index.PageURL)
	message.Acknowledge(ctx)
	c.pageIntakeObserver.PageIndexed(ctx, message.Identity(), index.PageURL)
}

func (c *OfferedPageConsumer) admitURLMetadata(
	ctx context.Context,
	message pullintake.PendingMessage,
	index pagerwi.PageRWI,
) bool {
	receipt, err := c.urlReceiver.Receive(ctx, []yacymodel.URLMetadata{index.Metadata})
	if err != nil {
		c.pageIntakeObserver.URLMetadataAdmissionFailed(
			ctx, message.Identity(), index.PageURL, err,
		)
		message.Return(ctx)

		return false
	}
	if receipt.Busy {
		c.pageIntakeObserver.URLMetadataAdmissionBusy(ctx, message.Identity(), index.PageURL)
		message.Return(ctx)

		return false
	}
	if len(receipt.ErrorURL) != 0 {
		c.pageIntakeObserver.URLMetadataAdmissionFailed(
			ctx,
			message.Identity(),
			index.PageURL,
			errURLMetadataAdmissionRejected,
		)
		message.Return(ctx)

		return false
	}
	c.pageIntakeObserver.URLMetadataAdmitted(ctx, message.Identity(), index.PageURL)

	return true
}

func (c *OfferedPageConsumer) admitPostings(
	ctx context.Context,
	message pullintake.PendingMessage,
	index pagerwi.PageRWI,
) bool {
	receipt, err := c.postingReceiver.Receive(ctx, index.Postings)
	if err != nil {
		c.pageIntakeObserver.PostingsAdmissionFailed(
			ctx,
			message.Identity(),
			index.PageURL,
			len(index.Postings),
			err,
		)
		message.Return(ctx)

		return false
	}
	if receipt.Busy {
		c.pageIntakeObserver.PostingsAdmissionBusy(
			ctx,
			message.Identity(),
			index.PageURL,
			len(index.Postings),
		)
		message.Return(ctx)

		return false
	}
	c.pageIntakeObserver.PostingsAdmitted(
		ctx,
		message.Identity(),
		index.PageURL,
		len(index.Postings),
	)

	return true
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
