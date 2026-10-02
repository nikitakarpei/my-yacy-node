package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func findingsAfterURLMetadataLookup(
	query searchquery.Query,
	answers *discoveryAnswers,
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) queryfindings.Findings {
	documentsThePeersSent := answers.joinedDocumentsThePeersSent()
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			documentsThePeersSent.KeepMetadataThePeerSent(metadata, answeredAsk.Ask.Peer.Hash)
		}
	}

	return findingsOf(
		query,
		documentsThePeersSent.FoundDocuments(),
		answers.amountOfDocumentsHeldPerQueryWord(),
	)
}

func findingsOf(
	query searchquery.Query,
	foundDocuments []queryfindings.FoundDocument,
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:                query.WordHashes(),
		CompoundWords:             query.CompoundWords,
		FoundDocuments:            foundDocuments,
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWord,
	}
}
