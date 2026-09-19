// Package applog reports the pages one query read to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

const msgPageReadingPerformed = "page reading performed"

type PageReadingLog struct{}

func (PageReadingLog) PageReadingPerformed(
	ctx context.Context,
	pageReading pagereading.PerformedPageReading,
) {
	slog.DebugContext(ctx, msgPageReadingPerformed,
		slog.Int("amountOfPagesToRead", pageReading.AmountOfPagesToRead),
		slog.Int("amountOfPagesRead", pageReading.AmountOfPagesRead),
		slog.Int("amountOfPagesUnreachable", pageReading.AmountOfPagesUnreachable),
		slog.Int("amountOfPagesRefused", pageReading.AmountOfPagesRefused),
		slog.Int("amountOfPagesGone", pageReading.AmountOfPagesGone),
		slog.Int("amountOfPagesUnreadable", pageReading.AmountOfPagesUnreadable),
		slog.Int(
			"amountOfPagesOfAnUnsupportedKind",
			pageReading.AmountOfPagesOfAnUnsupportedKind,
		),
		slog.Int("amountOfPagesOutOfBudget", pageReading.AmountOfPagesOutOfBudget),
		slog.Duration("timeSpent", pageReading.TimeSpent),
		slog.Duration("timeSpentFetching", pageReading.TimeSpentFetching),
		slog.Duration("timeSpentReading", pageReading.TimeSpentReading),
	)
}
