package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords: queryWords,
		FoundDocuments: foundDocumentsFrom(
			answeredAsks, queryWords, chosenPeersPerQueryWord.ReliabilityPerPeer(),
		),
	}
}

func foundDocumentsFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
	reliabilityOfEachPeer map[yacymodel.Hash]float64,
) []queryanswers.FoundDocument {
	countedWord, countedWordIsKnown := wordThePeersCountedFor(queryWords)

	var foundDocuments []queryanswers.FoundDocument
	placeOfEachDocument := map[yacymodel.URLHash]int{}
	postingsPerDocument := map[yacymodel.URLHash]*queryanswers.PostingsOfOneDocumentAcrossReplicas{}
	for _, answeredAsk := range answeredAsks {
		for placeGivenByThePeer, matchedDocument := range answeredAsk.MatchedDocuments {
			place, alreadyFound := placeOfEachDocument[matchedDocument.Metadata.Hash]
			if !alreadyFound {
				place = len(foundDocuments)
				placeOfEachDocument[matchedDocument.Metadata.Hash] = place
				postingsPerDocument[matchedDocument.Metadata.Hash] = &queryanswers.PostingsOfOneDocumentAcrossReplicas{}
				foundDocuments = append(
					foundDocuments, queryanswers.FoundDocumentFrom(matchedDocument.Metadata),
				)
			}
			foundDocuments[place].PlacesGivenByPeers = append(
				foundDocuments[place].PlacesGivenByPeers,
				queryanswers.PlaceGivenByPeer{
					Peer:                 answeredAsk.Ask.Peer.Hash,
					ReliabilityOfThePeer: reliabilityOfEachPeer[answeredAsk.Ask.Peer.Hash],
					Place:                placeGivenByThePeer,
				},
			)
			if !countedWordIsKnown {
				continue
			}
			postingsPerDocument[matchedDocument.Metadata.Hash].Take(
				countedWord, matchedDocument.Posting,
			)
		}
	}

	return foundDocumentsCountedAcrossTheReplicas(foundDocuments, postingsPerDocument)
}

func wordThePeersCountedFor(queryWords []yacymodel.Hash) (yacymodel.Hash, bool) {
	if len(queryWords) != 1 {
		return yacymodel.Hash{}, false
	}

	return queryWords[0], true
}

func foundDocumentsCountedAcrossTheReplicas(
	foundDocuments []queryanswers.FoundDocument,
	postingsPerDocument map[yacymodel.URLHash]*queryanswers.PostingsOfOneDocumentAcrossReplicas,
) []queryanswers.FoundDocument {
	for place, foundDocument := range foundDocuments {
		postings := postingsPerDocument[foundDocument.Hash]
		foundDocuments[place].HitsPerQueryWord = postings.HitsPerQueryWord()
		foundDocuments[place].AmountOfWords = postings.AmountOfWords()
		foundDocuments[place].LinkCounts = postings.LinkCounts()
	}

	return foundDocuments
}
