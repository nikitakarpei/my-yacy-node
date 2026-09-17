package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedMatchedAndHeldDocumentsRound struct {
	AmountOfQueryWords                                     int
	AmountOfQueryWordsHeldByNoPeer                         int
	AmountOfFullyListedQueryWords                          int
	AmountOfPeersAskedForMatchedAndHeldDocuments           int
	AmountOfPeersThatAnsweredMatchedAndHeldDocuments       int
	AmountOfPeersThatListedADocument                       int
	LeadingQueryWordStanding                               LeadingQueryWordStanding
	AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord int
	AmountOfMatchedDocumentsAcrossAnswers                  int
	AmountOfMatchedDocumentsCountedByAPeer                 int
	AmountOfDocumentsHeldInEachAnswer                      []int
}

func performedMatchedAndHeldDocumentsRoundFrom(
	round matchedAndHeldDocumentsRound,
) PerformedMatchedAndHeldDocumentsRound {
	return PerformedMatchedAndHeldDocumentsRound{
		AmountOfQueryWords: len(round.queryWords),
		AmountOfQueryWordsHeldByNoPeer: amountOfQueryWordsHeldByNoPeerAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		AmountOfFullyListedQueryWords: amountOfFullyListedQueryWordsAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		AmountOfPeersAskedForMatchedAndHeldDocuments: amountOfPeersAcross(
			round.asks,
			peerOfMatchedAndHeldDocumentsAsk,
		),
		AmountOfPeersThatAnsweredMatchedAndHeldDocuments: amountOfPeersAcross(
			round.answeredAsks, peerOfAnsweredMatchedAndHeldDocumentsAsk,
		),
		AmountOfPeersThatListedADocument: amountOfPeersAcross(
			answeredAsksWithAListedDocument(round.answeredAsks),
			peerOfAnsweredMatchedAndHeldDocumentsAsk,
		),
		LeadingQueryWordStanding: leadingQueryWordStandingAmong(
			round.queryWordsFewestDocumentsFirst,
		),
		AmountOfDocumentsListedByThePeersOfTheLeadingQueryWord: len(
			round.leadingQueryWord().documentsListed(),
		),
		AmountOfMatchedDocumentsAcrossAnswers: amountOfMatchedDocumentsAcrossAnswers(
			round.answeredAsks,
		),
		AmountOfMatchedDocumentsCountedByAPeer: amountOfMatchedDocumentsCountedByAPeer(
			round.answeredAsks,
		),
		AmountOfDocumentsHeldInEachAnswer: amountOfDocumentsHeldInEachAnswer(round.answeredAsks),
	}
}

func amountOfQueryWordsHeldByNoPeerAmong(queryWords []queryWordAcrossReplicas) int {
	amount := 0
	for _, queryWord := range queryWords {
		if len(queryWord.documentsListed()) > 0 {
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

func peerOfMatchedAndHeldDocumentsAsk(
	ask peerasks.MatchedAndHeldDocumentsAsk,
) peerdirectory.AskablePeer {
	return ask.Peer
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
