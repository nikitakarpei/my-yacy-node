package wordjoined

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedWordJoinedSpread struct {
	AmountOfQueryWords                     int
	AmountOfQueryWordsNoPeerHeld           int
	AmountOfPeersAsked                     int
	AmountOfPeersThatAnswered              int
	AmountOfPeersHoldingAQueryWord         int
	AmountOfJoinedDocuments                int
	AmountOfJoinedDocumentsWithMetadata    int
	AmountOfMatchedDocumentsAcrossAnswers  int
	AmountOfMatchedDocumentsCountedByAPeer int
	AmountOfDocumentsHeldInEachAnswer      []int
	AmountOfDocumentsAskedMetadataFor      int
	AmountOfAskedDocumentsWithMetadata     int
	TimeSpent                              time.Duration
}

//nolint:revive // argument-limit: the stages one word joined spread passes
func performedWordJoinedSpreadFrom(
	queryWords []yacymodel.Hash,
	heldDocumentsAsks []peerasks.HeldDocumentsAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
	documentsWithoutMetadata map[yacymodel.URLHash]struct{},
	urlMetadataAsks []peerasks.URLMetadataAsk,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	timeSpent time.Duration,
) PerformedWordJoinedSpread {
	asksWithAHeldDocument := answeredAsksWithAHeldDocument(answeredHeldDocumentsAsks)

	return PerformedWordJoinedSpread{
		AmountOfQueryWords: len(queryWords),
		AmountOfQueryWordsNoPeerHeld: len(queryWords) -
			amountOfWordsAcrossAnsweredAsks(asksWithAHeldDocument),
		AmountOfPeersAsked: amountOfPeersAcrossAsks(heldDocumentsAsks),
		AmountOfPeersThatAnswered: amountOfPeersAcrossAnsweredAsks(
			answeredHeldDocumentsAsks,
		),
		AmountOfPeersHoldingAQueryWord: amountOfPeersAcrossAnsweredAsks(
			asksWithAHeldDocument,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsWithMetadata: len(joinedDocuments) -
			len(documentsWithoutMetadata),
		AmountOfMatchedDocumentsAcrossAnswers: amountOfMatchedDocumentsAcrossAnswers(
			answeredHeldDocumentsAsks,
		),
		AmountOfMatchedDocumentsCountedByAPeer: amountOfMatchedDocumentsCountedByAPeer(
			answeredHeldDocumentsAsks,
		),
		AmountOfDocumentsHeldInEachAnswer: amountOfDocumentsHeldInEachAnswer(
			answeredHeldDocumentsAsks,
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
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []peerasks.AnsweredHeldDocumentsAsk {
	kept := make([]peerasks.AnsweredHeldDocumentsAsk, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.DocumentsHeldForTheWord) == 0 {
			continue
		}
		kept = append(kept, answeredAsk)
	}

	return kept
}

func amountOfWordsAcrossAnsweredAsks(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	words := map[yacymodel.Hash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		words[answeredAsk.Ask.Word] = struct{}{}
	}

	return len(words)
}

func amountOfPeersAcrossAsks(asks []peerasks.HeldDocumentsAsk) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, ask := range asks {
		peers[ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfPeersAcrossAnsweredAsks(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answeredAsk := range answeredAsks {
		peers[answeredAsk.Ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfMatchedDocumentsAcrossAnswers(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		amount += len(answeredAsk.MatchedDocuments)
	}

	return amount
}

func amountOfMatchedDocumentsCountedByAPeer(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
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
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []int {
	documentsHeld := make([]int, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		documentsHeldForTheWord, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		documentsHeld = append(documentsHeld, documentsHeldForTheWord)
	}

	return documentsHeld
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
