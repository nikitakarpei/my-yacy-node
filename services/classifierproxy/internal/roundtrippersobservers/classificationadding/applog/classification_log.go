// Package applog writes the facts of adding classifications to pages to the
// application log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/classificationadding"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

type ClassificationLog struct{}

func (ClassificationLog) PageClassified(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	classification pageclassification.Classification,
) {
	slog.DebugContext(ctx, "page classified",
		slog.String("address", address.String()),
		slog.Group("classification", assessmentAttributesOf(classification)...),
	)
}

func assessmentAttributesOf(classification pageclassification.Classification) []any {
	attributes := make([]any, 0, len(classification))
	for category, assessment := range classification {
		attributes = append(attributes, slog.Group(string(category),
			slog.String("verdict", assessment.Verdict().String()),
			slog.Float64("score", assessment.Score),
		))
	}
	return attributes
}

func (ClassificationLog) ReadingSlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	slog.DebugContext(ctx, "reading slot waited",
		slog.String("address", address.String()),
		slog.Duration("slotWait", slotWait),
	)
}

func (ClassificationLog) ClassificationSkipped(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason classificationadding.SkipReason,
) {
	slog.DebugContext(ctx, "classification skipped",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (ClassificationLog) PageRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason classificationadding.RefusalReason,
) {
	slog.WarnContext(ctx, "page refused",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (ClassificationLog) BodyPrefixReadFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, "body prefix read failed",
		slog.String("address", address.String()),
		slog.Any("error", cause),
	)
}
