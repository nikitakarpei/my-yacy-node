// Package applog reports to the service log where the query word document amounts
// failed.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	msgDocumentAmountLookupFailed = "remembered query word document amount could not be read"
	msgDocumentAmountStoreFailed  = "query word document amount could not be remembered"
)

type QueryWordDocumentAmountsLog struct{}

func (QueryWordDocumentAmountsLog) DocumentAmountLookupFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	slog.WarnContext(ctx, msgDocumentAmountLookupFailed,
		slog.String("word", word.String()),
		slog.Any("error", err),
	)
}

func (QueryWordDocumentAmountsLog) DocumentAmountStoreFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	slog.WarnContext(ctx, msgDocumentAmountStoreFailed,
		slog.String("word", word.String()),
		slog.Any("error", err),
	)
}
