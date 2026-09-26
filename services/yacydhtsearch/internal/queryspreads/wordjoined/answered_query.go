package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func answeredQueryFrom(
	query searchquery.Query,
	discoveryRound discoveryRound,
	urlMetadataLookupRound urlMetadataLookupRound,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:     discoveryRound.queryWords,
		CompoundWords:  query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(urlMetadataLookupRound.answeredAsks),
		DocumentsHeldPerQueryWord: discoveryRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) []queryanswers.FoundDocument {
	documentsThePeersSent := queryanswers.EmptyDocumentsThePeersSent()
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			documentsThePeersSent.KeepMetadataThePeerSent(
				metadata, answeredAsk.Ask.Peer.Hash,
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}
