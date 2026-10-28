package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataLookupOfJoinedDocuments struct {
	listings          *documentholders.Listings
	found             *documentsTheSpreadFound
	urlMetadataLookup *urlmetadataasks.Lookup
}

func (lookup urlMetadataLookupOfJoinedDocuments) DocumentsJoined(documents yacymodel.URLHashes) {
	lookup.urlMetadataLookup.AskFor(
		lookup.listings.HoldersOf(lookup.found.withoutMetadataAmong(documents)),
	)
}
