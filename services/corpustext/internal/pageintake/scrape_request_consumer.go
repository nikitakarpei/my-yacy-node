// Package pageintake derives the readable text of each page the crawler scrapeRequest and indexes it.
package pageintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/scrapedpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

const (
	msgScrapeFailed  = "scrape request scrape failed"
	msgIndexFailed   = "scrape request index failed"
	msgPageIndexed   = "scrape request indexed"
	msgNoTextDerived = "scrape request derives no text, nothing indexed"
)

type PageScraper interface {
	Scrape(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		targetFormat documentextraction.Format,
	) (pagescrape.ScrapedPage, bool, error)
}

type SearchIndex interface {
	Index(ctx context.Context, document searchdocument.Document) error
}

type IndexProgress interface {
	PageReceived()
	PageIndexed()
	ScrapeFailed()
	IndexFailed()
	IndexObserved(elapsed time.Duration)
}

type ScrapeRequestConsumer struct {
	source      pullintake.MessageSource
	scraper     PageScraper
	searchIndex SearchIndex
	progress    IndexProgress
	concurrency int
}

func NewScrapeRequestConsumer(
	source pullintake.MessageSource,
	scraper PageScraper,
	searchIndex SearchIndex,
	progress IndexProgress,
	concurrency int,
) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:      source,
		scraper:     scraper,
		searchIndex: searchIndex,
		progress:    progress,
		concurrency: concurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	c.progress.PageReceived()
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	scrapedAt := time.Now()
	scraped, derived, err := c.scraper.Scrape(
		ctx, scrapeRequest.CanonicalURL, documentextraction.FormatReadableText,
	)
	if err != nil {
		slog.WarnContext(ctx, msgScrapeFailed,
			slog.String("url", scrapeRequest.CanonicalURL.String()),
			slog.Any("error", err),
		)
		c.progress.ScrapeFailed()
		_ = msg.Nak()
		return nil
	}
	if !derived {
		slog.DebugContext(
			ctx,
			msgNoTextDerived,
			slog.String("url", scrapeRequest.CanonicalURL.String()),
		)
		_ = msg.Ack()
		return nil
	}
	return c.index(ctx, msg, scrapedpagedocument.Of(scraped, scrapedAt))
}

func (c *ScrapeRequestConsumer) index(
	ctx context.Context,
	msg jetstream.Msg,
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
		_ = msg.Nak()
		return nil
	}
	c.progress.PageIndexed()
	slog.DebugContext(ctx, msgPageIndexed, slog.String("url", document.URL))
	_ = msg.Ack()
	return nil
}
