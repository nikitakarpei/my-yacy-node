package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataLookupOfJoinedDocuments struct {
	holders           documentholders.Holders
	urlMetadataLookup *urlmetadataasks.Lookup
}

func (lookup urlMetadataLookupOfJoinedDocuments) DocumentsJoined(documents yacymodel.URLHashes) {
	lookup.urlMetadataLookup.AskFor(
		lookup.holders.HoldersOf(lookup.holders.WithoutMetadataAmong(documents)),
	)
}
