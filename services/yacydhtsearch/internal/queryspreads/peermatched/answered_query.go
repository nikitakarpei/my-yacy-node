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
			documentsThePeersSent.KeepDocumentThePeerMatched(
				answeredAsk.Ask.Peer.Hash,
				answeredAsk.Ask.Word,
				matchedDocument.Metadata,
				postingOfTheWordIn(answeredAsk, matchedDocument),
			)
		}
	}

	return documentsThePeersSent.FoundDocuments()
}

func postingOfTheWordIn(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
	matchedDocument peerasks.MatchedDocument,
) yacymodel.Optional[yacymodel.RWIPosting] {
	if len(answeredAsk.Ask.OtherWordsToMatch) > 0 {
		return yacymodel.None[yacymodel.RWIPosting]()
	}

	return matchedDocument.Posting
}
