package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	query searchquery.Query,
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	urlMetadataLookupRound urlMetadataLookupRound,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:    discoveryRound.queryWords,
		CompoundWords: query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(
			discoveryRound, joinedDocuments, urlMetadataLookupRound,
		),
		DocumentsHeldPerQueryWord: discoveryRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	discoveryRound discoveryRound,
	joinedDocuments distinctDocuments,
	urlMetadataLookupRound urlMetadataLookupRound,
) []queryanswers.FoundDocument {
	documentsThePeersSent := queryanswers.EmptyDocumentsThePeersSent()
	for _, answeredAsk := range discoveryRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			keepTheDocumentThePeerMatched(
				documentsThePeersSent,
				matchedDocument,
				answeredAsk.Ask.Peer.Hash,
				yacymodel.Some(answeredAsk.Ask.Word),
			)
		}
	}
	for _, answeredAsk := range urlMetadataLookupRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			documentsThePeersSent.KeepMetadataThePeerSent(
				metadata, answeredAsk.Ask.Peer.Hash,
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}

func keepTheDocumentThePeerMatched(
	documentsThePeersSent *queryanswers.DocumentsThePeersSent,
	matchedDocument peerasks.MatchedDocument,
	peer yacymodel.Hash,
	word yacymodel.Optional[yacymodel.Hash],
) {
	documentsThePeersSent.KeepMetadataThePeerSent(matchedDocument.Metadata, peer)
	posting, sent := matchedDocument.Posting.Get()
	if !sent {
		return
	}
	documentsThePeersSent.KeepPostingThePeerSent(
		matchedDocument.Metadata.Hash, peer, word, posting,
	)
}
