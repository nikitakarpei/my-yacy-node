// Package scraperequestintake scrapes each page the crawl fleet scrapeRequest and stores its
// reverse word index: the page's URL metadata, then its postings.
package scraperequestintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	msgScrapeFailed   = "scrape request scrape failed"
	msgNoIndexDerived = "scrape request derives no index, nothing stored"
	msgPageStored     = "scrape request stored"
	msgStoreDeferred  = "scrape request store deferred"
)

type PageScraper interface {
	Scrape(ctx context.Context, pageURL string) (pagescrape.ScrapedPage, bool, error)
}

type ScrapeRequestConsumer struct {
	source      pullintake.MessageSource
	scraper     PageScraper
	urls        urlmeta.URLReceiver
	postings    rwiadmission.PostingReceiver
	concurrency int
}

type Config struct {
	Source      pullintake.MessageSource
	Scraper     PageScraper
	URLs        urlmeta.URLReceiver
	Postings    rwiadmission.PostingReceiver
	Concurrency int
}

func NewScrapeRequestConsumer(config Config) *ScrapeRequestConsumer {
	return &ScrapeRequestConsumer{
		source:      config.Source,
		scraper:     config.Scraper,
		urls:        config.URLs,
		postings:    config.Postings,
		concurrency: config.Concurrency,
	}
}

func (c *ScrapeRequestConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *ScrapeRequestConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	reachedAt := time.Now()
	scraped, derived, err := c.scraper.Scrape(ctx, scrapeRequest.CanonicalURL.String())
	if err != nil {
		slog.WarnContext(ctx, msgScrapeFailed,
			slog.String("url", scrapeRequest.CanonicalURL.String()),
			slog.Any("error", err),
		)
		_ = msg.Nak()
		return nil
	}
	if !derived {
		slog.DebugContext(
			ctx,
			msgNoIndexDerived,
			slog.String("url", scrapeRequest.CanonicalURL.String()),
		)
		_ = msg.Ack()
		return nil
	}
	c.store(ctx, msg, pagerwi.Of(scraped, reachedAt))
	return nil
}

func (c *ScrapeRequestConsumer) store(
	ctx context.Context,
	msg jetstream.Msg,
	index pagerwi.PageRWI,
) {
	urlReceipt, err := c.urls.Receive(ctx, []yacymodel.URLMetadata{index.Metadata})
	if err != nil || urlReceipt.Busy {
		redeliver(ctx, msg, index.CanonicalURL.String(), err)
		return
	}
	postingReceipt, err := c.postings.Receive(ctx, index.Postings)
	if err != nil || postingReceipt.Busy {
		redeliver(ctx, msg, index.CanonicalURL.String(), err)
		return
	}
	_ = msg.Ack()
	slog.DebugContext(ctx, msgPageStored,
		slog.String("url", index.CanonicalURL.String()),
		slog.Int("postings", len(index.Postings)),
	)
}

func redeliver(ctx context.Context, msg jetstream.Msg, canonicalURL string, cause error) {
	slog.WarnContext(ctx, msgStoreDeferred,
		slog.String("url", canonicalURL),
		slog.Any("error", cause),
	)
	_ = msg.Nak()
}
