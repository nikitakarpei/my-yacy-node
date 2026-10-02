// Package applog reports the pages one query read to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

const (
	msgPageReadingPerformed         = "page reading performed"
	msgPageReadingRunFinished       = "page reading run finished"
	msgPagesReadAheadAfterTheFinish = "pages read ahead after the finish"
	msgPagesReadAfterTheFinish      = "pages read after the finish"
)

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
		slog.Int("amountOfPagesRefusingIndexing", pageReading.AmountOfPagesRefusingIndexing),
		slog.Int("amountOfPagesUnreadable", pageReading.AmountOfPagesUnreadable),
		slog.Int(
			"amountOfPagesOfAnUnsupportedKind",
			pageReading.AmountOfPagesOfAnUnsupportedKind,
		),
		slog.Int("amountOfPagesOutOfBudget", pageReading.AmountOfPagesOutOfBudget),
		slog.Int("amountOfPagesCutOff", pageReading.AmountOfPagesCutOff),
		slog.Int(
			"amountOfPagesReadAssessedAsSpam",
			pageReading.AmountOfPagesReadPerSpamVerdict[spamassessment.Spam],
		),
		slog.Int(
			"amountOfPagesReadAssessedAsClean",
			pageReading.AmountOfPagesReadPerSpamVerdict[spamassessment.Clean],
		),
		slog.Int(
			"amountOfPagesReadUnassessed",
			pageReading.AmountOfPagesReadPerSpamVerdict[spamassessment.Unassessed],
		),
		slog.Duration("timeSpent", pageReading.TimeSpent),
		slog.Duration("timeSpentFetching", pageReading.TimeSpentFetching),
		slog.Duration("timeSpentReading", pageReading.TimeSpentReading),
	)
}

func (PageReadingLog) PageReadingRunFinished(
	ctx context.Context,
	run pagereading.FinishedPageReadingRun,
) {
	slog.DebugContext(ctx, msgPageReadingRunFinished,
		slog.Int("amountOfPagesUnwanted", run.AmountOfPagesUnwanted),
	)
}

func (PageReadingLog) PagesReadAheadAfterTheFinish(ctx context.Context, amountOfPagesToRead int) {
	slog.WarnContext(ctx, msgPagesReadAheadAfterTheFinish,
		slog.Int("amountOfPagesToRead", amountOfPagesToRead),
	)
}

func (PageReadingLog) PagesReadAfterTheFinish(ctx context.Context, amountOfPagesWanted int) {
	slog.WarnContext(ctx, msgPagesReadAfterTheFinish,
		slog.Int("amountOfPagesWanted", amountOfPagesWanted),
	)
}
