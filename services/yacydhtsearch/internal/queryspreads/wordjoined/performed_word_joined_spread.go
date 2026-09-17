package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSpread struct {
	AmountOfQueryWords                                int
	AmountOfQueryWordsHeldByNoPeer                    int
	AmountOfShortQueryWords                           int
	AmountOfPeersAsked                                int
	AmountOfPeersThatAnswered                         int
	AmountOfPeersHoldingAQueryWord                    int
	AmountOfAnchorDocuments                           int
	AmountOfDocumentsPastTheHeldDocumentsCeiling      int
	AmountOfPeersAskedForHeldDocuments                int
	AmountOfPeersThatAnsweredHeldDocuments            int
	AmountOfEmptyHeldDocumentsAnswers                 int
	AmountOfJoinedDocumentsBeforeTheHeldDocumentsAsks int
	AmountOfJoinedDocuments                           int
	AmountOfJoinedDocumentsWithMetadata               int
	AmountOfMatchedDocumentsAcrossAnswers             int
	AmountOfMatchedDocumentsCountedByAPeer            int
	AmountOfDocumentsHeldInEachAnswer                 []int
	AmountOfDocumentsAskedMetadataFor                 int
	AmountOfAskedDocumentsWithMetadata                int
	TimeSpent                                         time.Duration
}

//nolint:revive // argument-limit: the stages one word joined spread passes
func performedWordJoinedSpreadFrom(
	queryWords []yacymodel.Hash,
	matchedAndHeldDocumentsAsks []peerasks.MatchedAndHeldDocumentsAsk,
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	anchor anchorOfTheQuery,
	heldDocuments heldDocumentsRound,
	joinedDocuments map[yacymodel.URLHash]struct{},
	documentsWithoutMetadata map[yacymodel.URLHash]struct{},
	urlMetadataAsks []peerasks.URLMetadataAsk,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	asksWithAHeldDocument := answeredAsksWithAHeldDocument(answeredMatchedAndHeldDocumentsAsks)

	return PerformedWordJoinedSpread{
		AmountOfShortQueryWords: heldDocuments.amountOfShortQueryWords,
		AmountOfAnchorDocuments: len(anchor.documents),
		AmountOfDocumentsPastTheHeldDocumentsCeiling: heldDocuments.
			amountOfDocumentsPastTheHeldDocumentsCeiling,
		AmountOfPeersAskedForHeldDocuments: amountOfPeersAcrossHeldDocumentsAsks(
			heldDocuments.asks,
		),
		AmountOfPeersThatAnsweredHeldDocuments: amountOfPeersAcrossAnsweredHeldDocumentsAsks(
			heldDocuments.answeredAsks,
		),
		AmountOfEmptyHeldDocumentsAnswers: amountOfEmptyHeldDocumentsAnswers(
			heldDocuments.answeredAsks,
		),
		AmountOfJoinedDocumentsBeforeTheHeldDocumentsAsks: len(joinedDocumentsOf(
			anchor,
			answeredMatchedAndHeldDocumentsAsks,
			nil,
			queryWords,
		)),
		AmountOfQueryWords: len(queryWords),
		AmountOfQueryWordsHeldByNoPeer: len(queryWords) -
			amountOfQueryWordsAcrossAnsweredAsks(asksWithAHeldDocument),
		AmountOfPeersAsked: amountOfPeersAcrossAsks(matchedAndHeldDocumentsAsks),
		AmountOfPeersThatAnswered: amountOfPeersAcrossAnsweredAsks(
			answeredMatchedAndHeldDocumentsAsks,
		),
		AmountOfPeersHoldingAQueryWord: amountOfPeersAcrossAnsweredAsks(
			asksWithAHeldDocument,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) -
			len(documentsWithoutMetadata),
		AmountOfMatchedDocumentsAcrossAnswers: amountOfMatchedDocumentsAcrossAnswers(
			answeredMatchedAndHeldDocumentsAsks,
		),
		AmountOfMatchedDocumentsCountedByAPeer: amountOfMatchedDocumentsCountedByAPeer(
			answeredMatchedAndHeldDocumentsAsks,
		),
		AmountOfDocumentsHeldInEachAnswer: amountOfDocumentsHeldInEachAnswer(
			answeredMatchedAndHeldDocumentsAsks,
		),
		AmountOfDocumentsAskedMetadataFor: amountOfDocumentsAskedMetadataFor(urlMetadataAsks),
		AmountOfAskedDocumentsWithMetadata: amountOfAskedDocumentsWithMetadata(
			urlMetadataAsks,
			answeredURLMetadataAsks,
		),
		TimeSpent: timeSpent,
	}
}

func answeredAsksWithAHeldDocument(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	keptAnsweredAsks := make([]peerasks.AnsweredMatchedAndHeldDocumentsAsk, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.DocumentsHeldForTheWord) == 0 {
			continue
		}
		keptAnsweredAsks = append(keptAnsweredAsks, answeredAsk)
	}

	return keptAnsweredAsks
}

func amountOfQueryWordsAcrossAnsweredAsks(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) int {
	queryWords := map[yacymodel.Hash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		queryWords[answeredAsk.Ask.Word] = struct{}{}
	}

	return len(queryWords)
}

func amountOfPeersAcrossAsks(asks []peerasks.MatchedAndHeldDocumentsAsk) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, ask := range asks {
		peers[ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfPeersAcrossAnsweredAsks(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answeredAsk := range answeredAsks {
		peers[answeredAsk.Ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfPeersAcrossHeldDocumentsAsks(asks []peerasks.HeldDocumentsAsk) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, ask := range asks {
		peers[ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfPeersAcrossAnsweredHeldDocumentsAsks(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answeredAsk := range answeredAsks {
		peers[answeredAsk.Ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfEmptyHeldDocumentsAnswers(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.DocumentsHeldForTheWord) > 0 {
			continue
		}
		amount++
	}

	return amount
}

func amountOfMatchedDocumentsAcrossAnswers(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		amount += len(answeredAsk.MatchedDocuments)
	}

	return amount
}

func amountOfMatchedDocumentsCountedByAPeer(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !matchedDocument.CountOfAWordTheAskNamed.CountedByAPeer() {
				continue
			}
			amount++
		}
	}

	return amount
}

func amountOfDocumentsHeldInEachAnswer(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []int {
	amountOfDocumentsHeldInEachAnswer := make([]int, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		amountOfDocumentsHeldForTheWord, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldInEachAnswer = append(
			amountOfDocumentsHeldInEachAnswer, amountOfDocumentsHeldForTheWord,
		)
	}

	return amountOfDocumentsHeldInEachAnswer
}

func amountOfDocumentsAskedMetadataFor(asks []peerasks.URLMetadataAsk) int {
	return len(documentsAcrossURLMetadataAsks(asks))
}

func documentsAcrossURLMetadataAsks(
	asks []peerasks.URLMetadataAsk,
) map[yacymodel.URLHash]struct{} {
	documents := map[yacymodel.URLHash]struct{}{}
	for _, ask := range asks {
		for _, document := range ask.Documents {
			documents[document] = struct{}{}
		}
	}

	return documents
}

func amountOfAskedDocumentsWithMetadata(
	asks []peerasks.URLMetadataAsk,
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) int {
	askedDocuments := documentsAcrossURLMetadataAsks(asks)
	documentsWithMetadata := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			if _, asked := askedDocuments[metadata.Hash]; !asked {
				continue
			}
			documentsWithMetadata[metadata.Hash] = struct{}{}
		}
	}

	return len(documentsWithMetadata)
}
