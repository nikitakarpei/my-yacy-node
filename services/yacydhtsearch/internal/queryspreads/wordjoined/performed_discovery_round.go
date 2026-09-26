package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
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
	answers := round.settledAsks.answers()

	return PerformedDiscoveryRound{
		AmountOfQueryWords:    len(round.queryWords),
		AmountOfCompoundWords: len(round.compoundWords),
		AmountOfQueryWordsHeldByNoPeer: amountOfQueryWordsHeldByNoPeerAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		SampledPartition:                   round.sampledPartition,
		AmountOfQueryWordsWithASample:      round.amountOfQueryWordsWithASample,
		AmountOfPeersWithANonEmptyAbstract: amountOfPeersWithANonEmptyAbstractAmong(answers),
		LeadingQueryWordChoice:             round.chosenLeadingQueryWord.choice,
		AmountOfDocumentsOfTheLeadingQueryWord: len(
			round.leadingQueryWord().documents(),
		),
		OtherWordAsksPerPartition:             round.otherWordAsksPerPartition,
		AmountOfMatchedDocumentsAcrossAnswers: amountOfMatchedDocumentsAcrossAnswers(answers),
		AmountOfMatchedDocumentsWithAPosting:  amountOfMatchedDocumentsWithAPosting(answers),
		AmountOfDocumentsHeldInEachAnswer:     amountOfDocumentsHeldInEachAnswer(answers),
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

func amountOfPeersWithANonEmptyAbstractAmong(answers []wordpartitionasks.ReplicaAnswer) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answer := range answers {
		if len(answer.ListedDocuments) == 0 {
			continue
		}
		peers[answer.Replica] = struct{}{}
	}

	return len(peers)
}

func amountOfMatchedDocumentsAcrossAnswers(answers []wordpartitionasks.ReplicaAnswer) int {
	amount := 0
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if !listedDocument.Metadata.Present() {
				continue
			}
			amount++
		}
	}

	return amount
}

func amountOfMatchedDocumentsWithAPosting(answers []wordpartitionasks.ReplicaAnswer) int {
	amount := 0
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if !listedDocument.Posting.Present() {
				continue
			}
			amount++
		}
	}

	return amount
}

func amountOfDocumentsHeldInEachAnswer(answers []wordpartitionasks.ReplicaAnswer) []int {
	amountOfDocumentsHeldInEachAnswer := make([]int, 0, len(answers))
	for _, answer := range answers {
		amountOfDocumentsHeldForTheWord, counted := answer.AmountOfDocumentsHeld.Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldInEachAnswer = append(
			amountOfDocumentsHeldInEachAnswer, amountOfDocumentsHeldForTheWord,
		)
	}

	return amountOfDocumentsHeldInEachAnswer
}
