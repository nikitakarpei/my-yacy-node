package markdownintake

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	msgMarkdownStoreFailed = "page markdown store failed"
	msgMarkdownStored      = "page markdown stored"
)

type PageMarkdownCorpus interface {
	Put(ctx context.Context, canonicalURL yacycrawlcontract.CanonicalURL, markdown []byte) error
}

type StoreProgress interface {
	PageReceived()
	PageStored()
	StoreFailed()
}

type PageMarkdownConsumer struct {
	source      pullintake.MessageSource
	corpus      PageMarkdownCorpus
	progress    StoreProgress
	concurrency int
}

func NewPageMarkdownConsumer(
	source pullintake.MessageSource,
	corpus PageMarkdownCorpus,
	progress StoreProgress,
	concurrency int,
) *PageMarkdownConsumer {
	return &PageMarkdownConsumer{
		source:      source,
		corpus:      corpus,
		progress:    progress,
		concurrency: concurrency,
	}
}

func (c *PageMarkdownConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.concurrency, c.processOne)
}

func (c *PageMarkdownConsumer) processOne(ctx context.Context, msg jetstream.Msg) error {
	c.progress.PageReceived()
	page, err := yacycrawlcontract.UnmarshalPageMarkdownRepresentation(msg.Data())
	if err != nil {
		return poisonhalt.Halt(ctx, msg, err)
	}
	if err := c.corpus.Put(ctx, page.CanonicalURL, page.Markdown); err != nil {
		slog.WarnContext(ctx, msgMarkdownStoreFailed,
			slog.String("url", page.CanonicalURL.String()),
			slog.Any("error", err),
		)
		c.progress.StoreFailed()
		_ = msg.Nak()
		return nil
	}
	c.progress.PageStored()
	slog.DebugContext(ctx, msgMarkdownStored, slog.String("url", page.CanonicalURL.String()))
	_ = msg.Ack()
	return nil
}
