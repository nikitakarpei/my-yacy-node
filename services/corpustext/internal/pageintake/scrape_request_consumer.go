// Package pageintake derives the readable text of each page the crawler scrapeRequest and indexes it.
package pageintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/scrapedpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

const (
	msgFetchFailed      = "scrape request fetch failed"
	msgFetchDeferred    = "scrape request fetch deferred by the origin"
	msgNothingToScrape  = "scrape request fetch holds no content to scrape"
	msgExtractionFailed = "scrape request document extraction failed, nothing indexed"
	msgIndexFailed      = "scrape request index failed"
	msgPageIndexed      = "scrape request indexed"
	msgNoTextDerived    = "scrape request derives no text, nothing indexed"
)

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) (pagefetch.FetchOutcome, error)
}

type SearchIndex interface {
	Index(ctx context.Context, document searchdocument.Document) error
}

type IndexProgress interface {
	ScrapeRequestReceived()
	PageIndexed()
	ScrapeFailed()
	IndexFailed()
	IndexObserved(elapsed time.Duration)
}

type ScrapeRequestConsumer struct {
	source                         pullintake.MessageSource
	fetcher                        PageFetcher
	formatDerivations              pageformats.FormatDerivationCatalog
	searchIndex                    SearchIndex
	progress                       IndexProgress
	scrapeRequestIntakeConcurrency int
}

type Config struct {
	Source                         pullintake.MessageSource
	Fetcher                        PageFetcher
	FormatDerivations              pageformats.FormatDerivationCatalog
	SearchIndex                    SearchIndex
	Progress                       IndexProgress
	ScrapeRequestIntakeConcurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:                         config.Source,
		fetcher:                        config.Fetcher,
		formatDerivations:              config.FormatDerivations,
		searchIndex:                    config.SearchIndex,
		progress:                       config.Progress,
		scrapeRequestIntakeConcurrency: config.ScrapeRequestIntakeConcurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.scrapeRequestIntakeConcurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	c.progress.ScrapeRequestReceived()
	scrapeRequest, err := pagescrapecontract.UnmarshalScrapeRequest(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	scrapedAt := time.Now()
	scrapedPage, scrapable := c.fetch(ctx, message, scrapeRequest)
	if !scrapable {
		return nil
	}
	document, extracted := c.documentOf(ctx, scrapedPage)
	if !extracted {
		message.Acknowledge(ctx)
		return nil
	}
	text, derived := c.readableTextOf(ctx, document, scrapedPage.LandedURL)
	if !derived {
		slog.DebugContext(ctx, msgNoTextDerived, slog.String("url", scrapedPage.PageURL.String()))
		message.Acknowledge(ctx)
		return nil
	}
	return c.index(
		ctx,
		message,
		scrapedpagedocument.Of(scrapedPage.PageURL, document, text, scrapedAt),
	)
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	message pullintake.PendingMessage,
	request pagescrapecontract.ScrapeRequest,
) (pagescrapecontract.OfferedPage, bool) {
	fetchURL := request.FetchURL
	outcome, err := c.fetcher.Fetch(ctx, fetchURL, pagefetch.PageVersion{})
	if err != nil {
		slog.WarnContext(ctx, msgFetchFailed,
			slog.String("url", fetchURL.String()),
			slog.Any("error", err),
		)
		c.progress.ScrapeFailed()
		message.Return(ctx)
		return pagescrapecontract.OfferedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return pagescrapecontract.OfferedPageFrom(request, outcome.Page), true
	case pagefetch.FetchFailed:
		slog.WarnContext(ctx, msgFetchFailed, slog.String("url", fetchURL.String()))
		c.progress.ScrapeFailed()
		message.Return(ctx)
	case pagefetch.FetchDeferred:
		slog.DebugContext(ctx, msgFetchDeferred,
			slog.String("url", fetchURL.String()),
			slog.Duration("deferFor", outcome.DeferFor),
		)
		message.ReturnAfter(ctx, outcome.DeferFor)
	default:
		slog.DebugContext(ctx, msgNothingToScrape, slog.String("url", fetchURL.String()))
		message.Acknowledge(ctx)
	}
	return pagescrapecontract.OfferedPage{}, false
}

func (c *ScrapeRequestConsumer) documentOf(
	ctx context.Context,
	scrapedPage pagescrapecontract.OfferedPage,
) (documentextraction.Document, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, scrapedPage.Body, scrapedPage.ContentType, scrapedPage.LandedURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", scrapedPage.LandedURL.String()),
			slog.Any("error", err),
		)
		return documentextraction.Document{}, false
	}
	return document, true
}

func (c *ScrapeRequestConsumer) readableTextOf(
	ctx context.Context,
	document documentextraction.Document,
	landedURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	return c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, document, landedURL,
	)
}

func (c *ScrapeRequestConsumer) index(
	ctx context.Context,
	message pullintake.PendingMessage,
	document searchdocument.Document,
) error {
	started := time.Now()
	err := c.searchIndex.Index(ctx, document)
	c.progress.IndexObserved(time.Since(started))
	if err != nil {
		slog.WarnContext(ctx, msgIndexFailed,
			slog.String("url", document.URL),
			slog.Any("error", err),
		)
		c.progress.IndexFailed()
		message.Return(ctx)
		return nil
	}
	c.progress.PageIndexed()
	slog.DebugContext(ctx, msgPageIndexed, slog.String("url", document.URL))
	message.Acknowledge(ctx)
	return nil
}
