// Package applog reports to the service log how many found documents of one
// query ranked lower because their host is not among the well-linked hosts.
package applog

import (
	"context"
	"log/slog"
)

const msgDocumentsDemoted = "documents outside the well-linked hosts were demoted"

type RelevanceLog struct{}

func (RelevanceLog) DocumentsDemoted(ctx context.Context, amountOfDemotedDocuments int) {
	slog.DebugContext(
		ctx,
		msgDocumentsDemoted,
		slog.Int("amountOfDemotedDocuments", amountOfDemotedDocuments),
	)
}
