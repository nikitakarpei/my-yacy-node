package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	query searchquery.Query,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:     query.WordHashes(),
		CompoundWords:  query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(answeredAsks, query.WordHashes()),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) []queryanswers.FoundDocument {
	documentsThePeersSent := queryanswers.EmptyDocumentsThePeersSent()
	countedWord := wordThePeersCountedFor(queryWords)
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			keepTheDocumentThePeerMatched(
				documentsThePeersSent, matchedDocument, answeredAsk.Ask.Peer.Hash, countedWord,
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}

func wordThePeersCountedFor(
	queryWords []yacymodel.Hash,
) yacymodel.Optional[yacymodel.Hash] {
	if len(queryWords) != 1 {
		return yacymodel.None[yacymodel.Hash]()
	}

	return yacymodel.Some(queryWords[0])
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
