package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
	query searchquery.Query,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:     query.WordHashes(),
		CompoundWords:  query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(answeredAsks),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) []queryanswers.FoundDocument {
	documentsThePeersSent := queryanswers.EmptyDocumentsThePeersSent()
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			keepTheDocumentThePeerMatched(
				documentsThePeersSent,
				matchedDocument,
				answeredAsk.Ask.Peer.Hash,
				answeredAsk.Ask.Word,
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}

func keepTheDocumentThePeerMatched(
	documentsThePeersSent *queryanswers.DocumentsThePeersSent,
	matchedDocument peerasks.MatchedDocument,
	peer yacymodel.Hash,
	word yacymodel.Hash,
) {
	documentsThePeersSent.KeepMetadataThePeerSent(matchedDocument.Metadata, peer)
	posting, sent := matchedDocument.Posting.Get()
	if !sent {
		return
	}
	documentsThePeersSent.KeepPostingThePeerSent(
		matchedDocument.Metadata.Hash, peer, yacymodel.Some(word), posting,
	)
}
