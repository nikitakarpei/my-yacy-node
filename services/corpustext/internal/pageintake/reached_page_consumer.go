// Package pageintake derives the readable text of each page the crawler reached and indexes it.
package pageintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/scrapedpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	msgScrapeFailed  = "reached page scrape failed"
	msgIndexFailed   = "reached page index failed"
	msgPageIndexed   = "reached page indexed"
	msgNoTextDerived = "reached page derives no text, nothing indexed"
)

type PageScraper interface {
	Scrape(ctx context.Context, pageURL string) (pagescrape.ScrapedPage, bool, error)
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

type ReachedPageConsumer struct {
	source      pullintake.MessageSource
	scraper     PageScraper
	searchIndex SearchIndex
	progress    IndexProgress
	concurrency int
}

func NewReachedPageConsumer(
	source pullintake.MessageSource,
	scraper PageScraper,
	searchIndex SearchIndex,
	progress IndexProgress,
	concurrency int,
) *ReachedPageConsumer {
	return &ReachedPageConsumer{
		source:      source,
		scraper:     scraper,
		searchIndex: searchIndex,
		progress:    progress,
		concurrency: concurrency,
	}
}

func (c *ReachedPageConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *ReachedPageConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	c.progress.PageReceived()
	reached, err := yacycrawlcontract.UnmarshalReachedPage(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	scrapedAt := time.Now()
	scraped, derived, err := c.scraper.Scrape(ctx, reached.CanonicalURL.String())
	if err != nil {
		slog.WarnContext(ctx, msgScrapeFailed,
			slog.String("url", reached.CanonicalURL.String()),
			slog.Any("error", err),
		)
		c.progress.ScrapeFailed()
		_ = msg.Nak()
		return nil
	}
	if !derived {
		slog.DebugContext(ctx, msgNoTextDerived, slog.String("url", reached.CanonicalURL.String()))
		_ = msg.Ack()
		return nil
	}
	return c.index(ctx, msg, scrapedpagedocument.Of(scraped, scrapedAt))
}

func (c *ReachedPageConsumer) index(
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
