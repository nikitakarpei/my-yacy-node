// Package applog reports to the service log where the document amounts of the
// query words came from.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
)

const (
	msgAmountsCountedFromReplicas = "document amounts counted from replicas"
	msgAmountsReadFromCache       = "document amounts read from cache"
)

type DocumentAmountsLog struct{}

func (DocumentAmountsLog) AmountsCountedFromReplicas(
	ctx context.Context,
	performed documentamounts.PerformedFromReplicas,
) {
	slog.DebugContext(ctx, msgAmountsCountedFromReplicas,
		slog.Uint64("partition", uint64(performed.Partition)),
		slog.Int("amountOfQueryWords", performed.AmountOfQueryWords),
		slog.Int("amountOfQueryWordsCounted", performed.AmountOfQueryWordsCounted),
	)
}

func (DocumentAmountsLog) AmountsReadFromCache(
	ctx context.Context,
	performed documentamounts.PerformedFromCache,
) {
	slog.DebugContext(ctx, msgAmountsReadFromCache,
		slog.Int("amountOfQueryWords", performed.AmountOfQueryWords),
		slog.Int("amountOfQueryWordsCached", performed.AmountOfQueryWordsCached),
		slog.Bool("allQueryWordsCached", performed.AllQueryWordsCached),
	)
}
