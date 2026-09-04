// Package pageintake derives the readable text of each page the scrape service offers,
// indexes it, and sends a receipt back for the caller waiting on that page.
package pageintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/scrapedpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

type SearchIndex interface {
	Index(ctx context.Context, document searchdocument.Document) error
}

type IntakeReceipts interface {
	ReportKeptPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ReportRejectedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type OfferedPageConsumer struct {
	source                     pullintake.MessageSource
	formatDerivations          pageformats.FormatDerivationCatalog
	searchIndex                SearchIndex
	intakeReceipts             IntakeReceipts
	pageIntakeObserver         PageIntakeObserver
	pageOfferIntakeConcurrency int
}

type Config struct {
	Source                     pullintake.MessageSource
	FormatDerivations          pageformats.FormatDerivationCatalog
	SearchIndex                SearchIndex
	IntakeReceipts             IntakeReceipts
	PageIntakeObserver         PageIntakeObserver
	PageOfferIntakeConcurrency int
}

func NewOfferedPageConsumer(config Config) *OfferedPageConsumer {
	return &OfferedPageConsumer{
		source:                     config.Source,
		formatDerivations:          config.FormatDerivations,
		searchIndex:                config.SearchIndex,
		intakeReceipts:             config.IntakeReceipts,
		pageIntakeObserver:         config.PageIntakeObserver,
		pageOfferIntakeConcurrency: config.PageOfferIntakeConcurrency,
	}
}

func (c *OfferedPageConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.pageOfferIntakeConcurrency, c.takeIn)
}

func (c *OfferedPageConsumer) takeIn(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	page, err := pagescrapecontract.UnmarshalOfferedPage(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	c.pageIntakeObserver.PageOffered(ctx, page.PageURL)
	indexedAt := time.Now()
	document, err := documentextraction.DocumentFrom(
		ctx, page.Body, page.ContentType, page.LandedURL,
	)
	if err != nil {
		c.pageIntakeObserver.NoDocumentExtracted(ctx, page.LandedURL, err)
		return c.reject(ctx, message, page.PageURL)
	}
	text, derived := c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, document, page.LandedURL,
	)
	if !derived {
		c.pageIntakeObserver.NoReadableTextDerived(ctx, page.PageURL)
		return c.reject(ctx, message, page.PageURL)
	}
	return c.index(
		ctx,
		message,
		page.PageURL,
		scrapedpagedocument.Of(page.PageURL, document, text, indexedAt),
	)
}

func (c *OfferedPageConsumer) index(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	document searchdocument.Document,
) error {
	started := time.Now()
	err := c.searchIndex.Index(ctx, document)
	c.pageIntakeObserver.IndexObserved(ctx, time.Since(started))
	if err != nil {
		c.pageIntakeObserver.IndexFailed(ctx, pageURL, err)
		message.Return(ctx)
		return nil
	}
	c.pageIntakeObserver.PageIndexed(ctx, pageURL)
	c.intakeReceipts.ReportKeptPage(ctx, pageURL)
	message.Acknowledge(ctx)
	return nil
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
