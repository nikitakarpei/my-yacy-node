package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type urlMetadataRound struct {
	documentsWithoutMetadata distinctDocuments
	asks                     []peerasks.URLMetadataAsk
	answeredAsks             []peerasks.AnsweredURLMetadataAsk
}

func documentsWithoutMetadataAmong(
	joinedDocuments distinctDocuments,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) distinctDocuments {
	documentsWithoutMetadata := maps.Clone(joinedDocuments)
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			delete(documentsWithoutMetadata, matchedDocument.Metadata.Hash)
		}
	}

	return documentsWithoutMetadata
}
