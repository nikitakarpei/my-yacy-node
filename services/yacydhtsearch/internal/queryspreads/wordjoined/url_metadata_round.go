package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataRound struct {
	documentsWithoutMetadata map[yacymodel.URLHash]struct{}
	asks                     []peerasks.URLMetadataAsk
	answeredAsks             []peerasks.AnsweredURLMetadataAsk
}

func documentsWithoutMetadataAmong(
	joinedDocuments map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) map[yacymodel.URLHash]struct{} {
	documentsWithoutMetadata := maps.Clone(joinedDocuments)
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			delete(documentsWithoutMetadata, matchedDocument.Metadata.Hash)
		}
	}

	return documentsWithoutMetadata
}
