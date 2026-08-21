// Package reachedpageintake scrapes each page the crawl fleet reached and stores its
// reverse word index: the page's URL metadata, then its postings.
package reachedpageintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	msgScrapeFailed     = "reached page scrape failed"
	msgNoIndexDerived   = "reached page derives no index, nothing stored"
	msgIndexUnbuildable = "reached page yields no reverse word index, page skipped"
	msgPageStored       = "reached page stored"
	msgStoreDeferred    = "reached page store deferred"
)

type PageScraper interface {
	Scrape(ctx context.Context, pageURL string) (pagescrape.ScrapedPage, bool, error)
}

type ReachedPageConsumer struct {
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

func NewReachedPageConsumer(config Config) *ReachedPageConsumer {
	return &ReachedPageConsumer{
		source:      config.Source,
		scraper:     config.Scraper,
		urls:        config.URLs,
		postings:    config.Postings,
		concurrency: config.Concurrency,
	}
}

func (c *ReachedPageConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *ReachedPageConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	reached, err := yacycrawlcontract.UnmarshalReachedPage(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	reachedAt := time.Now()
	scraped, derived, err := c.scraper.Scrape(ctx, reached.CanonicalURL)
	if err != nil {
		slog.WarnContext(ctx, msgScrapeFailed,
			slog.String("url", reached.CanonicalURL),
			slog.Any("error", err),
		)
		_ = msg.Nak()
		return nil
	}
	if !derived {
		slog.DebugContext(ctx, msgNoIndexDerived, slog.String("url", reached.CanonicalURL))
		_ = msg.Ack()
		return nil
	}
	index, err := pagerwi.Of(scraped, reachedAt)
	if err != nil {
		slog.WarnContext(ctx, msgIndexUnbuildable,
			slog.String("url", scraped.CanonicalURL),
			slog.Any("error", err),
		)
		_ = msg.Ack()
		return nil
	}
	c.store(ctx, msg, index)
	return nil
}

func (c *ReachedPageConsumer) store(
	ctx context.Context,
	msg jetstream.Msg,
	index pagerwi.PageRWI,
) {
	urlReceipt, err := c.urls.Receive(ctx, []yacymodel.URLMetadata{index.Metadata})
	if err != nil || urlReceipt.Busy {
		redeliver(ctx, msg, index.CanonicalURL, err)
		return
	}
	postingReceipt, err := c.postings.Receive(ctx, index.Postings)
	if err != nil || postingReceipt.Busy {
		redeliver(ctx, msg, index.CanonicalURL, err)
		return
	}
	_ = msg.Ack()
	slog.DebugContext(ctx, msgPageStored,
		slog.String("url", index.CanonicalURL),
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
