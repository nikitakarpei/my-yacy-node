package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedDiscoveryRound struct {
	AmountOfQueryWords                     int
	AmountOfCompoundWords                  int
	AmountOfQueryWordsHeldByNoPeer         int
	SampledPartition                       uint
	AmountOfQueryWordsWithASample          int
	AmountOfPeersWithANonEmptyAbstract     int
	LeadingQueryWordChoice                 LeadingQueryWordChoice
	AmountOfDocumentsOfTheLeadingQueryWord int
	OtherWordAsksPerPartition              map[uint]OtherWordAsks
	AmountOfMatchedDocumentsAcrossAnswers  int
	AmountOfMatchedDocumentsWithAPosting   int
	AmountOfDocumentsHeldInEachAnswer      []int
}

func performedDiscoveryRoundFrom(
	round discoveryRound,
) PerformedDiscoveryRound {
	return PerformedDiscoveryRound{
		AmountOfQueryWords:    len(round.queryWords),
		AmountOfCompoundWords: len(round.compoundWords),
		AmountOfQueryWordsHeldByNoPeer: amountOfQueryWordsHeldByNoPeerAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		SampledPartition:              round.sampledPartition,
		AmountOfQueryWordsWithASample: round.amountOfQueryWordsWithASample,
		AmountOfPeersWithANonEmptyAbstract: amountOfPeersWithANonEmptyAbstractAmong(
			round.answeredAsks,
		),
		LeadingQueryWordChoice: round.chosenLeadingQueryWord.choice,
		AmountOfDocumentsOfTheLeadingQueryWord: len(
			round.leadingQueryWord().documents(),
		),
		OtherWordAsksPerPartition: round.otherWordAsksPerPartition,
		AmountOfMatchedDocumentsAcrossAnswers: amountOfMatchedDocumentsAcrossAnswers(
			round.answeredAsks,
		),
		AmountOfMatchedDocumentsWithAPosting: amountOfMatchedDocumentsWithAPosting(
			round.answeredAsks,
		),
		AmountOfDocumentsHeldInEachAnswer: amountOfDocumentsHeldInEachAnswer(round.answeredAsks),
	}
}

func amountOfQueryWordsHeldByNoPeerAmong(queryWords []queryWordAcrossReplicas) int {
	amount := 0
	for _, queryWord := range queryWords {
		if len(queryWord.documents()) > 0 {
			continue
		}
		amount++
	}

	return amount
}

func amountOfPeersWithANonEmptyAbstractAmong(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.Abstract) == 0 {
			continue
		}
		peers[answeredAsk.Ask.Peer] = struct{}{}
	}

	return len(peers)
}

func amountOfMatchedDocumentsAcrossAnswers(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		amount += len(answeredAsk.MatchedDocuments)
	}

	return amount
}

func amountOfMatchedDocumentsWithAPosting(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		for _, matchedDocument := range answeredAsk.MatchedDocuments {
			if !matchedDocument.Posting.Present() {
				continue
			}
			amount++
		}
	}

	return amount
}

func amountOfDocumentsHeldInEachAnswer(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
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
