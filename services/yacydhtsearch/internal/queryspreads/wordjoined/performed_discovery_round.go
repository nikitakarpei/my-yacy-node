package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedDiscoveryRound struct {
	AmountOfQueryWords                            int
	AmountOfCompoundWords                         int
	AmountOfQueryWordsHeldByNoPeer                int
	AmountOfQueryWordsWithCompleteAbstracts       int
	AmountOfPeersWithANonEmptyAbstract            int
	LeadingQueryWordChoice                        LeadingQueryWordChoice
	AmountOfDocumentsOfTheLeadingQueryWord        int
	AmountOfPartitionsWithABetterLeadingQueryWord int
	AmountOfMatchedDocumentsAcrossAnswers         int
	AmountOfMatchedDocumentsWithAPosting          int
	AmountOfDocumentsHeldInEachAnswer             []int
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
		AmountOfQueryWordsWithCompleteAbstracts: amountOfQueryWordsWithCompleteAbstractsAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		AmountOfPeersWithANonEmptyAbstract: amountOfPeersAcross(
			answeredAsksWithANonEmptyAbstract(round.answeredAsks),
			peerOfAnsweredDiscoveryAsk,
		),
		LeadingQueryWordChoice: leadingQueryWordChoiceOf(round),
		AmountOfDocumentsOfTheLeadingQueryWord: len(
			round.leadingQueryWord().documents(),
		),
		AmountOfPartitionsWithABetterLeadingQueryWord: amountOfPartitionsWithABetterLeadingQueryWordIn(
			round,
		),
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

func amountOfQueryWordsWithCompleteAbstractsAmong(queryWords []queryWordAcrossReplicas) int {
	amount := 0
	for _, queryWord := range queryWords {
		if !queryWord.hasCompleteAbstracts() {
			continue
		}
		amount++
	}

	return amount
}

func peerOfAnsweredDiscoveryAsk(
	answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) peerdirectory.AskablePeer {
	return answeredAsk.Ask.Peer
}

func answeredAsksWithANonEmptyAbstract(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	keptAnsweredAsks := make([]peerasks.AnsweredMatchedAndHeldDocumentsAsk, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.Abstract) == 0 {
			continue
		}
		keptAnsweredAsks = append(keptAnsweredAsks, answeredAsk)
	}

	return keptAnsweredAsks
}

func amountOfPartitionsWithABetterLeadingQueryWordIn(
	round discoveryRound,
) int {
	queryWordsBesideTheLeadingQueryWord := round.queryWordsBesideTheLeadingQueryWord()
	amount := 0
	for partition, wordPartition := range round.leadingQueryWord().wordPartitions() {
		if wordPartition.hasACompleteAbstract() {
			continue
		}
		if !slices.ContainsFunc(
			queryWordsBesideTheLeadingQueryWord,
			func(queryWord queryWordAcrossReplicas) bool {
				return queryWord.wordPartitions()[partition].hasACompleteAbstract()
			},
		) {
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

func amountOfMatchedDocumentsWithAPosting(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
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
