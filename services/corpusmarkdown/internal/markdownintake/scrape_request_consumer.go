// Package markdownintake derives the markdown of each page the crawler scrapeRequest and stores it.
package markdownintake

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

const (
	msgFetchFailed         = "scrape request fetch failed"
	msgFetchDeferred       = "scrape request fetch deferred by the origin"
	msgNothingToScrape     = "scrape request fetch holds no content to scrape"
	msgExtractionFailed    = "scrape request document extraction failed, nothing stored"
	msgMarkdownStoreFailed = "page markdown store failed"
	msgMarkdownStored      = "page markdown stored"
	msgNoMarkdownDerived   = "scrape request derives no markdown, nothing stored"
)

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		version pagefetch.PageVersion,
	) (pagefetch.FetchOutcome, error)
}

type PageMarkdownCorpus interface {
	Put(ctx context.Context, canonicalURL canonicalurl.CanonicalURL, markdown []byte) error
}

type StoreProgress interface {
	ScrapeRequestReceived()
	PageStored()
	ScrapeFailed()
	StoreFailed()
}

type ScrapeRequestConsumer struct {
	source                         pullintake.MessageSource
	fetcher                        PageFetcher
	formatDerivations              pageformats.FormatDerivationCatalog
	corpus                         PageMarkdownCorpus
	progress                       StoreProgress
	scrapeRequestIntakeConcurrency int
}

type Config struct {
	Source                         pullintake.MessageSource
	Fetcher                        PageFetcher
	FormatDerivations              pageformats.FormatDerivationCatalog
	Corpus                         PageMarkdownCorpus
	Progress                       StoreProgress
	ScrapeRequestIntakeConcurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:                         config.Source,
		fetcher:                        config.Fetcher,
		formatDerivations:              config.FormatDerivations,
		corpus:                         config.Corpus,
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
	fetched, scrapable := c.fetch(ctx, message, scrapeRequest.CanonicalURL)
	if !scrapable {
		return nil
	}
	markdown, derived := c.markdownOf(ctx, fetched)
	if !derived {
		slog.DebugContext(ctx, msgNoMarkdownDerived,
			slog.String("url", fetched.FinalURL.String()),
		)
		message.Acknowledge(ctx)
		return nil
	}
	return c.store(ctx, message, fetched.FinalURL, markdown)
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

func (c *ScrapeRequestConsumer) markdownOf(
	ctx context.Context,
	fetched pagefetch.FetchedPage,
) ([]byte, bool) {
	document, err := documentextraction.DocumentFrom(
		ctx, fetched.Body, fetched.ContentType, fetched.FinalURL,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", fetched.FinalURL.String()),
			slog.Any("error", err),
		)
		return nil, false
	}
	markdown, derived := c.formatDerivations.BodyIn(
		ctx, documentextraction.FormatMarkdown, document, fetched.FinalURL,
	)
	return markdown, derived
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	message pullintake.PendingMessage,
	pageURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if err := c.corpus.Put(ctx, pageURL, markdown); err != nil {
		slog.WarnContext(ctx, msgMarkdownStoreFailed,
			slog.String("url", pageURL.String()),
			slog.Any("error", err),
		)
		c.progress.StoreFailed()
		message.Return(ctx)
		return nil
	}
	c.progress.PageStored()
	slog.DebugContext(ctx, msgMarkdownStored, slog.String("url", pageURL.String()))
	message.Acknowledge(ctx)
	return nil
}
