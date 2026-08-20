package crawlresults

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	msgIngestChunkAbsorbed = "ingest chunk absorbed"
	msgIngestChunkDeferred = "ingest chunk deferred"
	msgIngestChunkTooLarge = "ingest chunk exceeds posting batch cap, deferred until operator intervenes"
	msgIngestChunkUnknown  = "ingest chunk kind is not absorbable, discarded"

	msgIngestAckFailed  = "ingest chunk ack failed"
	msgIngestNakFailed  = "ingest chunk nak failed"
	msgIngestTermFailed = "ingest chunk term failed"
)

func (c *IngestConsumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-c.stream.Receive():
			if !ok {
				return
			}
			c.absorb(ctx, delivery)
		}
	}
}

func (c *IngestConsumer) absorb(ctx context.Context, delivery IngestDelivery) {
	switch chunk := delivery.Chunk.(type) {
	case yacycrawlcontract.PageRWIMetadataChunk:
		c.absorbMetadata(ctx, delivery, chunk)
	case yacycrawlcontract.PageRWIPostingChunk:
		c.absorbPostings(ctx, delivery, chunk)
	default:
		c.discard(ctx, delivery)
	}
}

func (c *IngestConsumer) discard(ctx context.Context, delivery IngestDelivery) {
	slog.ErrorContext(ctx, msgIngestChunkUnknown,
		slog.String("chunk", fmt.Sprintf("%T", delivery.Chunk)))
	if err := delivery.Term(ctx); err != nil {
		slog.WarnContext(ctx, msgIngestTermFailed, slog.Any("error", err))
	}
}

func (c *IngestConsumer) absorbMetadata(
	ctx context.Context,
	delivery IngestDelivery,
	chunk yacycrawlcontract.PageRWIMetadataChunk,
) {
	receipt, err := c.urls.Receive(ctx, chunk.Metadata)
	if err != nil || receipt.Busy {
		c.redeliver(ctx, delivery, chunk.CanonicalURL, err)
		return
	}
	c.acknowledge(ctx, delivery, chunk.CanonicalURL, len(chunk.Metadata), 0)
}

func (c *IngestConsumer) absorbPostings(
	ctx context.Context,
	delivery IngestDelivery,
	chunk yacycrawlcontract.PageRWIPostingChunk,
) {
	receipt, err := c.postings.Receive(ctx, chunk.Postings)
	if receipt.TooLarge {
		c.redeliverTooLarge(ctx, delivery, chunk.CanonicalURL, len(chunk.Postings))
		return
	}
	if err != nil || receipt.Busy {
		c.redeliver(ctx, delivery, chunk.CanonicalURL, err)
		return
	}
	c.acknowledge(ctx, delivery, chunk.CanonicalURL, 0, len(chunk.Postings))
}

func (c *IngestConsumer) acknowledge(
	ctx context.Context,
	delivery IngestDelivery,
	canonicalURL yacycrawlcontract.CanonicalURL,
	metadata int,
	postings int,
) {
	if err := delivery.Ack(ctx); err != nil {
		slog.WarnContext(ctx, msgIngestAckFailed,
			slog.String("url", canonicalURL.String()), slog.Any("error", err))
		return
	}
	slog.DebugContext(ctx, msgIngestChunkAbsorbed,
		slog.String("url", canonicalURL.String()),
		slog.Int("metadata", metadata),
		slog.Int("postings", postings))
}

func (c *IngestConsumer) redeliver(
	ctx context.Context,
	delivery IngestDelivery,
	canonicalURL yacycrawlcontract.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgIngestChunkDeferred,
		slog.String("url", canonicalURL.String()), slog.Any("error", cause))
	if err := delivery.Nak(ctx); err != nil {
		slog.WarnContext(ctx, msgIngestNakFailed,
			slog.String("url", canonicalURL.String()), slog.Any("error", err))
	}
}

func (c *IngestConsumer) redeliverTooLarge(
	ctx context.Context,
	delivery IngestDelivery,
	canonicalURL yacycrawlcontract.CanonicalURL,
	postingCount int,
) {
	slog.ErrorContext(ctx, msgIngestChunkTooLarge,
		slog.String("url", canonicalURL.String()), slog.Int("postings", postingCount))
	if err := delivery.Nak(ctx); err != nil {
		slog.WarnContext(ctx, msgIngestNakFailed,
			slog.String("url", canonicalURL.String()), slog.Any("error", err))
	}
}
