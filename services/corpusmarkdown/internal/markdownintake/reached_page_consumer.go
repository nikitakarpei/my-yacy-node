// Package markdownintake derives the markdown of each page the crawler reached and stores it.
package markdownintake

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	msgScrapeFailed        = "reached page scrape failed"
	msgMarkdownStoreFailed = "page markdown store failed"
	msgMarkdownStored      = "page markdown stored"
	msgNoMarkdownDerived   = "reached page derives no markdown, nothing stored"
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

type ReachedPageConsumer struct {
	source      pullintake.MessageSource
	scraper     PageScraper
	corpus      PageMarkdownCorpus
	progress    StoreProgress
	concurrency int
}

func NewReachedPageConsumer(
	source pullintake.MessageSource,
	scraper PageScraper,
	corpus PageMarkdownCorpus,
	progress StoreProgress,
	concurrency int,
) *ReachedPageConsumer {
	return &ReachedPageConsumer{
		source:      source,
		scraper:     scraper,
		corpus:      corpus,
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
		slog.DebugContext(
			ctx,
			msgNoMarkdownDerived,
			slog.String("url", reached.CanonicalURL.String()),
		)
		_ = msg.Ack()
		return nil
	}
	return c.store(ctx, msg, scraped)
}

func (c *ReachedPageConsumer) store(
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
