// Package applog reports to the service log how each partition of a word joined
// spread was asked for the other words.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
)

const msgAskedPartitionForTheOtherWords = "partition asked for the other words"

type DocumentAsksLog struct{}

func (DocumentAsksLog) AskedPartitionFor(
	ctx context.Context,
	partition uint,
	kind documentasks.Kind,
) {
	slog.DebugContext(ctx, msgAskedPartitionForTheOtherWords,
		slog.Uint64("partition", uint64(partition)),
		slog.String("otherWordAsks", string(kind)),
	)
}
