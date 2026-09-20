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
	return queryanswers.AnsweredQuery{
		QueryWords:       queryWords,
		FoundDocuments:   foundDocumentsFrom(answeredAsks),
		FactsPerDocument: factsPerDocumentFrom(answeredAsks, queryWords),
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

func factsPerDocumentFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) queryanswers.FactsPerDocument {
	factsPerDocument := queryanswers.FactsPerDocument{}
	countedWord, countedWordIsKnown := wordThePeersCountedFor(queryWords)
	if !countedWordIsKnown {
		return factsPerDocument
	}
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			factsPerDocument.KeepTheFirstPostingOfTheWord(
				matchedDocument.Metadata.Hash, countedWord, matchedDocument.Posting,
			)
		}
	}

	return factsPerDocument
}

func wordThePeersCountedFor(queryWords []yacymodel.Hash) (yacymodel.Hash, bool) {
	if len(queryWords) != 1 {
		return yacymodel.Hash{}, false
	}

	return queryWords[0], true
}
