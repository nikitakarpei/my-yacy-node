// Package applog reports to the service log how the URL metadata lookup of a word
// joined spread performed.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
)

const msgURLMetadataLookupPerformed = "url metadata lookup performed"

type URLMetadataLookupLog struct{}

func (URLMetadataLookupLog) URLMetadataLookupPerformed(
	ctx context.Context,
	lookup urlmetadataasks.Performed,
) {
	attributes := []slog.Attr{
		slog.Int("amountOfLookedUpDocuments", lookup.AmountOfLookedUpDocuments),
		slog.Int(
			"amountOfLookedUpDocumentsWithMetadata",
			lookup.AmountOfLookedUpDocumentsWithMetadata,
		),
		slog.Int("amountOfDocumentsNotAsked", lookup.AmountOfDocumentsNotAsked),
		slog.String("urlMetadataLookupEndReason", string(lookup.EndReason)),
		slog.Int("amountOfDocumentsCutOffDuringLookup", lookup.AmountOfDocumentsCutOff),
	}
	if timeToFirstAsk, asked := lookup.TimeToFirstAsk.Get(); asked {
		attributes = append(
			attributes, slog.Duration("timeToFirstUrlMetadataAsk", timeToFirstAsk),
		)
	}
	slog.LogAttrs(ctx, slog.LevelDebug, msgURLMetadataLookupPerformed, attributes...)
}
