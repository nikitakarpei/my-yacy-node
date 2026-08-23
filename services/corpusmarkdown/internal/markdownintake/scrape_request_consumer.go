// Package markdownintake derives the markdown of each page the crawler scrapeRequest and stores it.
package markdownintake

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

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
	source      pullintake.MessageSource
	fetcher     PageFetcher
	derivations pageformats.FormatDerivations
	corpus      PageMarkdownCorpus
	progress    StoreProgress
	concurrency int
}

type Config struct {
	Source      pullintake.MessageSource
	Fetcher     PageFetcher
	Derivations pageformats.FormatDerivations
	Corpus      PageMarkdownCorpus
	Progress    StoreProgress
	Concurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:      config.Source,
		fetcher:     config.Fetcher,
		derivations: config.Derivations,
		corpus:      config.Corpus,
		progress:    config.Progress,
		concurrency: config.Concurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	c.progress.ScrapeRequestReceived()
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	fetched, scrapable := c.fetch(ctx, msg, scrapeRequest.CanonicalURL)
	if !scrapable {
		return nil
	}
	markdown, derived := c.markdownOf(ctx, fetched)
	if !derived {
		slog.DebugContext(ctx, msgNoMarkdownDerived,
			slog.String("url", fetched.FinalURL.String()),
		)
		_ = msg.Ack()
		return nil
	}
	return c.store(ctx, msg, fetched.FinalURL, markdown)
}

func (c *ScrapeRequestConsumer) fetch(
	ctx context.Context,
	msg jetstream.Msg,
	pageURL canonicalurl.CanonicalURL,
) (pagefetch.FetchedPage, bool) {
	outcome, err := c.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil {
		slog.WarnContext(ctx, msgFetchFailed,
			slog.String("url", pageURL.String()),
			slog.Any("error", err),
		)
		c.progress.ScrapeFailed()
		_ = msg.Nak()
		return pagefetch.FetchedPage{}, false
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		return outcome.Page, true
	case pagefetch.FetchFailed:
		slog.WarnContext(ctx, msgFetchFailed, slog.String("url", pageURL.String()))
		c.progress.ScrapeFailed()
		_ = msg.Nak()
	case pagefetch.FetchDeferred:
		slog.DebugContext(ctx, msgFetchDeferred,
			slog.String("url", pageURL.String()),
			slog.Duration("deferFor", outcome.DeferFor),
		)
		_ = msg.NakWithDelay(outcome.DeferFor)
	default:
		slog.DebugContext(ctx, msgNothingToScrape, slog.String("url", pageURL.String()))
		_ = msg.Ack()
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
	markdown, derived, err := c.derivations.
		ForPage(document, fetched.FinalURL).
		Resolve(documentextraction.FormatMarkdown)
	if err != nil {
		slog.WarnContext(ctx, msgNoMarkdownDerived,
			slog.String("url", fetched.FinalURL.String()),
			slog.Any("error", err),
		)
		return nil, false
	}
	return markdown, derived
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	msg jetstream.Msg,
	pageURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if err := c.corpus.Put(ctx, pageURL, markdown); err != nil {
		slog.WarnContext(ctx, msgMarkdownStoreFailed,
			slog.String("url", pageURL.String()),
			slog.Any("error", err),
		)
		c.progress.StoreFailed()
		_ = msg.Nak()
		return nil
	}
	c.progress.PageStored()
	slog.DebugContext(ctx, msgMarkdownStored, slog.String("url", pageURL.String()))
	_ = msg.Ack()
	return nil
}
