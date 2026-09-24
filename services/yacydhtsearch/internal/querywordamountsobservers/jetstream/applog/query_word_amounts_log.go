// Package applog reports to the service log where the query word amounts
// failed.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	msgAmountLookupFailed = "remembered query word amount could not be read"
	msgAmountStoreFailed  = "query word amount could not be remembered"
)

type QueryWordAmountsLog struct{}

func (QueryWordAmountsLog) AmountLookupFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	slog.WarnContext(ctx, msgAmountLookupFailed,
		slog.String("word", word.String()),
		slog.Any("error", err),
	)
}

func (QueryWordAmountsLog) AmountStoreFailed(ctx context.Context, word yacymodel.Hash, err error) {
	slog.WarnContext(ctx, msgAmountStoreFailed,
		slog.String("word", word.String()),
		slog.Any("error", err),
	)
}
