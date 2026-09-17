// Package applog reports the shared history of probe answers to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
)

const (
	msgProbeAnswerAppendFailed = "the probe answer of a peer could not be appended to the history"
	msgProbeAnswerUnreadable   = "an answer in the history could not be read"
	msgProbeAnswerHistoryEnded = "the history of probe answers ended"
)

type ProbeAnswerHistoryLog struct{}

func (ProbeAnswerHistoryLog) ProbeAnswerAppendFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgProbeAnswerAppendFailed, slog.Any("error", err))
}

func (ProbeAnswerHistoryLog) ProbeAnswerUnreadable(
	ctx context.Context,
	position probeanswerhistory.ProbeAnswerPosition,
	err error,
) {
	slog.WarnContext(ctx, msgProbeAnswerUnreadable,
		slog.Uint64("position", uint64(position)),
		slog.Any("error", err),
	)
}

func (ProbeAnswerHistoryLog) ProbeAnswerHistoryEnded(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgProbeAnswerHistoryEnded, slog.Any("error", err))
}
