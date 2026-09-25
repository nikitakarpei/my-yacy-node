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
	AmountOfDocumentsInEachAbstract        []int
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
		OtherWordAsksPerPartition:       round.otherWordAsksPerPartition,
		AmountOfDocumentsInEachAbstract: amountOfDocumentsInEachAbstractOf(round.answeredAsks),
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
	answeredAsks []peerasks.AnsweredWordAbstractAsk,
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

func amountOfDocumentsInEachAbstractOf(
	answeredAsks []peerasks.AnsweredWordAbstractAsk,
) []int {
	amountOfDocumentsInEachAbstract := make([]int, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		amountOfDocumentsInEachAbstract = append(
			amountOfDocumentsInEachAbstract, len(answeredAsk.Abstract),
		)
	}

	return amountOfDocumentsInEachAbstract
}
