package urlmetadatacourier

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type boundedCourier struct {
	inner     Courier
	batchSize int
}

func NewBounded(inner Courier, batchSize int) Courier {
	return boundedCourier{inner: inner, batchSize: batchSize}
}

func (c boundedCourier) Deliver(
	ctx context.Context,
	recipient yacymodel.Seed,
	metadata []yacymodel.URLMetadata,
) Receipt {
	if len(metadata) == 0 {
		return Receipt{Outcome: Accepted}
	}

	var rejectedURLs []yacymodel.URLHash
	for start := 0; start < len(metadata); start += c.batchSize {
		end := min(start+c.batchSize, len(metadata))

		receipt := c.inner.Deliver(ctx, recipient, metadata[start:end])
		if receipt.Outcome != Accepted {
			if skipped := len(metadata) - end; skipped > 0 {
				slog.WarnContext(
					ctx,
					"url metadata batch delivery aborted after batch failure",
					slog.String("peer", recipient.Hash.String()),
					slog.String("outcome", string(receipt.Outcome)),
					slog.Int("deliveredUrls", start),
					slog.Int("skippedUrls", skipped),
				)
			}

			return Receipt{Outcome: receipt.Outcome}
		}

		rejectedURLs = append(rejectedURLs, receipt.URLsRejected...)
	}

	return Receipt{Outcome: Accepted, URLsRejected: rejectedURLs}
}
