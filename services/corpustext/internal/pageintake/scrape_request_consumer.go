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
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
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
	formats                        pageformats.DerivableFormats
	searchIndex                    SearchIndex
	progress                       IndexProgress
	scrapeRequestIntakeConcurrency int
}

type Config struct {
	Source                         pullintake.MessageSource
	Fetcher                        PageFetcher
	Formats                        pageformats.DerivableFormats
	SearchIndex                    SearchIndex
	Progress                       IndexProgress
	ScrapeRequestIntakeConcurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:                         config.Source,
		fetcher:                        config.Fetcher,
		formats:                        config.Formats,
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
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	scrapedAt := time.Now()
	fetched, scrapable := c.fetch(ctx, message, scrapeRequest.CanonicalURL)
	if !scrapable {
		return nil
	}
	document, text, derived := c.readableTextOf(ctx, fetched)
	if !derived {
		slog.DebugContext(ctx, msgNoTextDerived, slog.String("url", fetched.FinalURL.String()))
		message.Acknowledge(ctx)
		return nil
	}
	return c.index(ctx, message, scrapedpagedocument.Of(
		fetched.FinalURL, document, text, scrapedAt,
	))
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
) (pagefetch.FetchedPage, bool) {
	outcome, err := c.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil {
		slog.WarnContext(ctx, msgFetchFailed,
			slog.String("url", pageURL.String()),
			slog.Any("error", err),
		)
		c.progress.ScrapeFailed()
		message.Return(ctx)
		return pagefetch.FetchedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return outcome.Page, true
	case pagefetch.FetchFailed:
		slog.WarnContext(ctx, msgFetchFailed, slog.String("url", pageURL.String()))
		c.progress.ScrapeFailed()
		message.Return(ctx)
	case pagefetch.FetchDeferred:
		slog.DebugContext(ctx, msgFetchDeferred,
			slog.String("url", pageURL.String()),
			slog.Duration("deferFor", outcome.DeferFor),
		)
		message.ReturnAfter(ctx, outcome.DeferFor)
	default:
		slog.DebugContext(ctx, msgNothingToScrape, slog.String("url", pageURL.String()))
		message.Acknowledge(ctx)
	}
	return pagefetch.FetchedPage{}, false
}

func (c *ScrapeRequestConsumer) readableTextOf(
	ctx context.Context,
	fetched pagefetch.FetchedPage,
) (documentextraction.Document, []byte, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, fetched.Body, fetched.ContentType, fetched.FinalURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", fetched.FinalURL.String()),
			slog.Any("error", err),
		)
		return documentextraction.Document{}, nil, false
	}
	text, derived, err := c.formats.BodyIn(
		documentextraction.FormatReadableText, document, fetched.FinalURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgNoTextDerived,
			slog.String("url", fetched.FinalURL.String()),
			slog.Any("error", err),
		)
		return documentextraction.Document{}, nil, false
	}
	return document, text, derived
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
