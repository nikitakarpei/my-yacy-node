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
		QueryWords:     queryWords,
		FoundDocuments: foundDocumentsFrom(answeredAsks, queryWords),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) []queryanswers.FoundDocument {
	countedWord, countedWordIsKnown := wordThePeersCountedFor(queryWords)

	var foundDocuments []queryanswers.FoundDocument
	placeOfEachDocument := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			place, alreadyFound := placeOfEachDocument[matchedDocument.Metadata.Hash]
			if !alreadyFound {
				place = len(foundDocuments)
				placeOfEachDocument[matchedDocument.Metadata.Hash] = place
				foundDocuments = append(
					foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
				)
			}
			if !countedWordIsKnown {
				continue
			}
			keepTheFirstPostingOfTheWord(
				&foundDocuments[place],
				countedWord,
				matchedDocument.Posting,
			)
		}
	}

	return foundDocuments
}

func wordThePeersCountedFor(queryWords []yacymodel.Hash) (yacymodel.Hash, bool) {
	if len(queryWords) != 1 {
		return yacymodel.Hash{}, false
	}

	return queryWords[0], true
}

func keepTheFirstPostingOfTheWord(
	foundDocument *queryanswers.FoundDocument,
	word yacymodel.Hash,
	posting yacymodel.Optional[yacymodel.RWIPosting],
) {
	sentPosting, sent := posting.Get()
	if !sent {
		return
	}
	if _, alreadyCounted := foundDocument.HitsPerQueryWord[word]; alreadyCounted {
		return
	}
	foundDocument.HitsPerQueryWord[word] = sentPosting.Hits
	foundDocument.AmountOfWordsAPeerCounted = mostWordsAnyPeerCounted(
		foundDocument.AmountOfWordsAPeerCounted, sentPosting.TextWords,
	)
	foundDocument.LinkCounts = yacymodel.Some(queryanswers.LinkCountsFrom(sentPosting))
}

func mostWordsAnyPeerCounted(
	amountOfWordsThePeersBeforeCounted yacymodel.Optional[int],
	amountOfWordsThisPeerCounted int,
) yacymodel.Optional[int] {
	if amountOfWordsThisPeerCounted <= 0 {
		return amountOfWordsThePeersBeforeCounted
	}

	return yacymodel.Some(
		max(amountOfWordsThePeersBeforeCounted.OrElse(0), amountOfWordsThisPeerCounted),
	)
}
