package rwidistribution

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type boundedURLMetadataCourier struct {
	inner     urlMetadataCourier
	batchSize int
}

func (c boundedURLMetadataCourier) Deliver(
	ctx context.Context,
	endpoint string,
	peer yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) urlMetadataReceipt {
	if len(metadata) == 0 {
		return urlMetadataReceipt{Outcome: urlMetadataAccepted}
	}

	var rejected []yacymodel.URLHash
	for start := 0; start < len(metadata); start += c.batchSize {
		end := min(start+c.batchSize, len(metadata))

		receipt := c.inner.Deliver(ctx, endpoint, peer, metadata[start:end])
		if receipt.Outcome != urlMetadataAccepted {
			if skipped := len(metadata) - end; skipped > 0 {
				slog.WarnContext(
					ctx,
					"url metadata batch delivery aborted after batch failure",
					slog.String("peer", peer.String()),
					slog.String("endpoint", endpoint),
					slog.String("outcome", string(receipt.Outcome)),
					slog.Int("deliveredUrls", start),
					slog.Int("skippedUrls", skipped),
				)
			}

			return urlMetadataReceipt{Outcome: receipt.Outcome}
		}

		rejected = append(rejected, receipt.URLsRejected...)
	}

	return urlMetadataReceipt{Outcome: urlMetadataAccepted, URLsRejected: rejected}
}
