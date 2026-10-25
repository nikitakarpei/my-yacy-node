package urlmetadatacourier

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type UnboundedCourier interface {
	Deliver(
		ctx context.Context,
		peer yacymodel.Hash,
		metadata []yacymodel.URLMetadata,
	) Receipt
}

type BoundedCourier struct {
	inner     UnboundedCourier
	batchSize int
}

func NewBounded(inner UnboundedCourier, batchSize int) BoundedCourier {
	return BoundedCourier{inner: inner, batchSize: batchSize}
}

func (c BoundedCourier) Deliver(
	ctx context.Context,
	peer yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) Receipt {
	var rejectedURLs []yacymodel.URLHash
	for start := 0; start < len(metadata); start += c.batchSize {
		end := min(start+c.batchSize, len(metadata))

		receipt := c.inner.Deliver(ctx, peer, metadata[start:end])
		if receipt.Outcome != Accepted {
			if skipped := len(metadata) - end; skipped > 0 {
				slog.WarnContext(
					ctx,
					"url metadata batch delivery aborted after batch failure",
					slog.String("peer", peer.String()),
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
