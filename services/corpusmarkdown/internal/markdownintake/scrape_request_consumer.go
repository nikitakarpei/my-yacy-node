// Package markdownintake derives the markdown of each page the crawler scrapeRequest and stores it.
package markdownintake

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

const (
	msgScrapeFailed        = "scrape request scrape failed"
	msgMarkdownStoreFailed = "page markdown store failed"
	msgMarkdownStored      = "page markdown stored"
	msgNoMarkdownDerived   = "scrape request derives no markdown, nothing stored"
)

type PageScraper interface {
	Scrape(ctx context.Context, pageURL string) (pagescrape.ScrapedPage, bool, error)
}

type PageMarkdownCorpus interface {
	Put(ctx context.Context, canonicalURL canonicalurl.CanonicalURL, markdown []byte) error
}

type StoreProgress interface {
	PageReceived()
	PageStored()
	ScrapeFailed()
	StoreFailed()
}

type ScrapeRequestConsumer struct {
	source      pullintake.MessageSource
	scraper     PageScraper
	corpus      PageMarkdownCorpus
	progress    StoreProgress
	concurrency int
}

func NewScrapeRequestConsumer(
	source pullintake.MessageSource,
	scraper PageScraper,
	corpus PageMarkdownCorpus,
	progress StoreProgress,
	concurrency int,
) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:      source,
		scraper:     scraper,
		corpus:      corpus,
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
	scraped, derived, err := c.scraper.Scrape(ctx, scrapeRequest.CanonicalURL.String())
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
			msgNoMarkdownDerived,
			slog.String("url", scrapeRequest.CanonicalURL.String()),
		)
		_ = msg.Ack()
		return nil
	}
	return c.store(ctx, msg, scraped)
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	msg jetstream.Msg,
	scraped pagescrape.ScrapedPage,
) error {
	if err := c.corpus.Put(ctx, scraped.CanonicalURL, scraped.Content); err != nil {
		slog.WarnContext(ctx, msgMarkdownStoreFailed,
			slog.String("url", scraped.CanonicalURL.String()),
			slog.Any("error", err),
		)
		c.progress.StoreFailed()
		_ = msg.Nak()
		return nil
	}
	c.progress.PageStored()
	slog.DebugContext(ctx, msgMarkdownStored, slog.String("url", scraped.CanonicalURL.String()))
	_ = msg.Ack()
	return nil
}
