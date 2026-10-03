// Package applog reports to the service log how the partitions of a word joined
// spread were asked for the other words.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
)

const msgPartitionsAskedForTheOtherWords = "partitions asked for the other words"

type DocumentAsksLog struct{}

func (DocumentAsksLog) AskedAmongTheDocuments(
	ctx context.Context,
	decisionPerPartition documentasks.DocumentsToMatchDecisionPerPartition,
) {
	slog.DebugContext(
		ctx,
		msgPartitionsAskedForTheOtherWords,
		slog.Any(
			"amountOfPartitionsPerOtherWordAsks",
			decisionPerPartition.AmountOfPartitionsPerDecision(),
		),
	)
}
