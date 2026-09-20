package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) queryanswers.AnsweredQuery {
	postingReplicasPerDocument := postingReplicasPerDocumentFrom(answeredAsks, queryWords)

	return queryanswers.AnsweredQuery{
		QueryWords:                 queryWords,
		FoundDocuments:             foundDocumentsFrom(answeredAsks),
		PostingReplicasPerDocument: postingReplicasPerDocument,
		FactsPerDocument:           queryanswers.FactsPerDocumentOf(postingReplicasPerDocument),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
) []queryanswers.FoundDocument {
	var foundDocuments []queryanswers.FoundDocument
	alreadyFoundDocuments := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if _, alreadyFound := alreadyFoundDocuments[matchedDocument.Metadata.Hash]; alreadyFound {
				continue
			}
			alreadyFoundDocuments[matchedDocument.Metadata.Hash] = struct{}{}
			foundDocuments = append(
				foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
			)
		}
	}

	return foundDocuments
}

func postingReplicasPerDocumentFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) queryanswers.PostingReplicasPerDocument {
	postingReplicasPerDocument := queryanswers.PostingReplicasPerDocument{}
	countedWord := wordThePeersCountedFor(queryWords)
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			posting, sent := matchedDocument.Posting.Get()
			if !sent {
				continue
			}
			postingReplicasPerDocument.Keep(
				matchedDocument.Metadata.Hash,
				queryanswers.PostingReplica{
					Holder:  answeredAsk.Ask.Peer.Hash,
					Word:    countedWord,
					Posting: posting,
				},
			)
		}
	}

	return postingReplicasPerDocument
}

func wordThePeersCountedFor(
	queryWords []yacymodel.Hash,
) yacymodel.Optional[yacymodel.Hash] {
	if len(queryWords) != 1 {
		return yacymodel.None[yacymodel.Hash]()
	}

	return yacymodel.Some(queryWords[0])
}
