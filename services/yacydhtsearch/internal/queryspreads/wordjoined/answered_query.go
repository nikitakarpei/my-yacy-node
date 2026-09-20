package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
) queryanswers.AnsweredQuery {
	postingReplicasPerDocument := postingReplicasPerDocumentFrom(
		matchedAndHeldDocumentsRound, joinedDocuments,
	)

	return queryanswers.AnsweredQuery{
		QueryWords: matchedAndHeldDocumentsRound.queryWords,
		FoundDocuments: foundDocumentsFrom(
			matchedAndHeldDocumentsRound, joinedDocuments, urlMetadataRound,
		),
		PostingReplicasPerDocument: postingReplicasPerDocument,
		MetadataPerDocument: metadataPerDocumentFrom(
			matchedAndHeldDocumentsRound, joinedDocuments, urlMetadataRound,
		),
		FactsPerDocument: queryanswers.FactsPerDocumentOf(postingReplicasPerDocument),
		DocumentsHeldPerQueryWord: matchedAndHeldDocumentsRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
) []queryanswers.FoundDocument {
	var foundDocuments []queryanswers.FoundDocument
	alreadyFoundDocuments := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			if _, alreadyFound := alreadyFoundDocuments[matchedDocument.Metadata.Hash]; alreadyFound {
				continue
			}
			alreadyFoundDocuments[matchedDocument.Metadata.Hash] = struct{}{}
			foundDocuments = append(
				foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
			)
		}
	}
	for _, answeredAsk := range urlMetadataRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if _, alreadyFound := alreadyFoundDocuments[metadata.Hash]; alreadyFound {
				continue
			}
			alreadyFoundDocuments[metadata.Hash] = struct{}{}
			foundDocuments = append(foundDocuments, queryanswers.FoundDocumentFrom(metadata))
		}
	}

	return foundDocuments
}

func metadataPerDocumentFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
	urlMetadataRound urlMetadataRound,
) queryanswers.MetadataPerDocument {
	metadataPerDocument := queryanswers.MetadataPerDocument{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			metadataPerDocument.Keep(matchedDocument.Metadata)
		}
	}
	for _, answeredAsk := range urlMetadataRound.answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			metadataPerDocument.Keep(metadata)
		}
	}

	return metadataPerDocument
}

func postingReplicasPerDocumentFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
) queryanswers.PostingReplicasPerDocument {
	postingReplicasPerDocument := queryanswers.PostingReplicasPerDocument{}
	for _, answeredAsk := range matchedAndHeldDocumentsRound.answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !joinedDocuments.contains(matchedDocument.Metadata.Hash) {
				continue
			}
			posting, sent := matchedDocument.Posting.Get()
			if !sent {
				continue
			}
			postingReplicasPerDocument.Keep(
				matchedDocument.Metadata.Hash,
				queryanswers.PostingReplica{
					Holder:  answeredAsk.Ask.Peer.Hash,
					Word:    yacymodel.Some(answeredAsk.Ask.Word),
					Posting: posting,
				},
			)
		}
	}

	return postingReplicasPerDocument
}
