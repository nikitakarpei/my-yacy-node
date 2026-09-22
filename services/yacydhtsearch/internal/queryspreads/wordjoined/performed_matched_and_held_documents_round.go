package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedMatchedAndHeldDocumentsRound struct {
	AmountOfQueryWords                                     int
	AmountOfCompoundWords                                  int
	AmountOfQueryWordsHeldByNoPeer                         int
	AmountOfFullyListedQueryWords                          int
	AmountOfPeersThatListedADocument                       int
	LeadingQueryWordChoice                                 LeadingQueryWordChoice
	AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord int
	AmountOfPartitionsWithABetterLeadingQueryWord          int
	AmountOfMatchedDocumentsAcrossAnswers                  int
	AmountOfMatchedDocumentsWithAPosting                   int
	AmountOfDocumentsHeldInEachAnswer                      []int
}

func performedMatchedAndHeldDocumentsRoundFrom(
	round matchedAndHeldDocumentsRound,
) PerformedMatchedAndHeldDocumentsRound {
	return PerformedMatchedAndHeldDocumentsRound{
		AmountOfQueryWords:    len(round.queryWords),
		AmountOfCompoundWords: len(round.compoundWords),
		AmountOfQueryWordsHeldByNoPeer: amountOfQueryWordsHeldByNoPeerAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		AmountOfFullyListedQueryWords: amountOfFullyListedQueryWordsAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		AmountOfPeersThatListedADocument: amountOfPeersAcross(
			answeredAsksWithAListedDocument(round.answeredAsks),
			peerOfAnsweredMatchedAndHeldDocumentsAsk,
		),
		LeadingQueryWordChoice: leadingQueryWordChoiceOf(round),
		AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord: len(
			round.leadingQueryWord().documentsListedByPeers(),
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
		if len(queryWord.documentsListedByPeers()) > 0 {
			continue
		}
		amount++
	}

	return amount
}

func amountOfFullyListedQueryWordsAmong(queryWords []queryWordAcrossReplicas) int {
	amount := 0
	for _, queryWord := range queryWords {
		if !queryWord.isFullyListed() {
			continue
		}
		amount++
	}

	return amount
}

func peerOfAnsweredMatchedAndHeldDocumentsAsk(
	answeredAsk peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) peerdirectory.AskablePeer {
	return answeredAsk.Ask.Peer
}

func answeredAsksWithAListedDocument(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	keptAnsweredAsks := make([]peerasks.AnsweredMatchedAndHeldDocumentsAsk, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.DocumentsListedForTheWord) == 0 {
			continue
		}
		keptAnsweredAsks = append(keptAnsweredAsks, answeredAsk)
	}

	return keptAnsweredAsks
}

func amountOfPartitionsWithABetterLeadingQueryWordIn(
	round matchedAndHeldDocumentsRound,
) int {
	queryWordsBesideTheLeadingQueryWord := round.queryWordsBesideTheLeadingQueryWord()
	amount := 0
	for partition, wordPartition := range round.leadingQueryWord().wordPartitions() {
		if wordPartition.isFullyListed() {
			continue
		}
		if !slices.ContainsFunc(
			queryWordsBesideTheLeadingQueryWord,
			func(queryWord queryWordAcrossReplicas) bool {
				return queryWord.wordPartitions()[partition].isFullyListed()
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
