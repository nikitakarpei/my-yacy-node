package pageintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	msgCrawledPageIndexFailed = "crawled page index failed"
	msgCrawledPageIndexed     = "crawled page indexed"
)

type SearchIndex interface {
	Index(ctx context.Context, page yacycrawlcontract.PageTextRepresentation) error
}

type IndexProgress interface {
	PageReceived()
	PageIndexed()
	IndexFailed()
	IndexObserved(elapsed time.Duration)
}

type CrawledPageConsumer struct {
	source      pullintake.MessageSource
	indexer     SearchIndex
	progress    IndexProgress
	concurrency int
}

func NewCrawledPageConsumer(
	source pullintake.MessageSource,
	indexer SearchIndex,
	progress IndexProgress,
	concurrency int,
) *CrawledPageConsumer {
	return &CrawledPageConsumer{
		source:      source,
		indexer:     indexer,
		progress:    progress,
		concurrency: concurrency,
	}
}

func (c *CrawledPageConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *CrawledPageConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	c.progress.PageReceived()
	page, err := yacycrawlcontract.UnmarshalPageTextRepresentation(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	started := time.Now()
	err = c.indexer.Index(ctx, page)
	c.progress.IndexObserved(time.Since(started))
	if err != nil {
		slog.WarnContext(ctx, msgCrawledPageIndexFailed,
			slog.String("url", page.CanonicalURL.String()),
			slog.Any("error", err),
		)
		c.progress.IndexFailed()
		_ = msg.Nak()
		return nil
	}
	c.progress.PageIndexed()
	slog.DebugContext(ctx, msgCrawledPageIndexed, slog.String("url", page.CanonicalURL.String()))
	_ = msg.Ack()
	return nil
}
