// Package pageintake derives the markdown of each page the scrape service offers, stores it
// under the URL the origin settled on, and sends a receipt back for the caller waiting on
// that page.
package pageintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

type PageMarkdownCorpus interface {
	Put(ctx context.Context, canonicalURL canonicalurl.CanonicalURL, markdown []byte) error
}

type IntakeReceipts interface {
	ReportKeptPage(ctx context.Context, pageURL canonicalurl.CanonicalURL) error
	ReportRejectedPage(ctx context.Context, pageURL canonicalurl.CanonicalURL) error
}

type OfferedPageConsumer struct {
	source                     pullintake.MessageSource
	formatDerivations          pageformats.FormatDerivationCatalog
	corpus                     PageMarkdownCorpus
	intakeReceipts             IntakeReceipts
	intakeProgress             IntakeProgress
	pageOfferIntakeConcurrency int
}

type Config struct {
	Source                     pullintake.MessageSource
	FormatDerivations          pageformats.FormatDerivationCatalog
	Corpus                     PageMarkdownCorpus
	IntakeReceipts             IntakeReceipts
	IntakeProgress             IntakeProgress
	PageOfferIntakeConcurrency int
}

func NewOfferedPageConsumer(config Config) *OfferedPageConsumer {
	return &OfferedPageConsumer{
		source:                     config.Source,
		formatDerivations:          config.FormatDerivations,
		corpus:                     config.Corpus,
		intakeReceipts:             config.IntakeReceipts,
		intakeProgress:             config.IntakeProgress,
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
	c.intakeProgress.PageOffered(ctx, page.PageURL)
	document, err := documentextraction.DocumentFrom(
		ctx, page.Body, page.ContentType, page.LandedURL,
	)
	if err != nil {
		c.intakeProgress.NoDocumentExtracted(ctx, page.LandedURL, err)
		return c.reject(ctx, message, page.PageURL)
	}
	markdown, derived := c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatMarkdown, document, page.LandedURL,
	)
	if !derived {
		c.intakeProgress.NoMarkdownDerived(ctx, page.PageURL)
		return c.reject(ctx, message, page.PageURL)
	}
	return c.store(ctx, message, page.PageURL, markdown)
}

func (c *OfferedPageConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if err := c.corpus.Put(ctx, pageURL, markdown); err != nil {
		c.intakeProgress.MarkdownNotStored(ctx, pageURL, err)
		message.Return(ctx)
		return nil
	}
	c.intakeProgress.MarkdownStored(ctx, pageURL)
	if err := c.intakeReceipts.ReportKeptPage(ctx, pageURL); err != nil {
		c.intakeProgress.IntakeReceiptNotSent(ctx, pageURL, err)
	}
	message.Acknowledge(ctx)
	return nil
}

func (c *OfferedPageConsumer) reject(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
) error {
	if err := c.intakeReceipts.ReportRejectedPage(ctx, pageURL); err != nil {
		c.intakeProgress.IntakeReceiptNotSent(ctx, pageURL, err)
	}
	message.Acknowledge(ctx)
	return nil
}
