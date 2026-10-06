// Package applog reports to the service log where the document amounts of the
// query words came from.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
)

const msgAmountsReadFromCache = "document amounts read from cache"

type DocumentAmountsLog struct{}

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
