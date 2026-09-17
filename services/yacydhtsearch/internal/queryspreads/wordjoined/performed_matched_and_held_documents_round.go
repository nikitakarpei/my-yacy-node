package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedMatchedAndHeldDocumentsRound struct {
	AmountOfQueryWords                     int
	AmountOfQueryWordsHeldByNoPeer         int
	AmountOfFullyListedQueryWords          int
	AmountOfPeersAsked                     int
	AmountOfPeersThatAnswered              int
	AmountOfPeersHoldingAQueryWord         int
	AmountOfAnchorDocuments                int
	AmountOfMatchedDocumentsAcrossAnswers  int
	AmountOfMatchedDocumentsCountedByAPeer int
	AmountOfDocumentsHeldInEachAnswer      []int
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
		AmountOfPeersAsked: amountOfPeersAcross(round.asks, peerOfMatchedAndHeldDocumentsAsk),
		AmountOfPeersThatAnswered: amountOfPeersAcross(
			round.answeredAsks, peerOfAnsweredMatchedAndHeldDocumentsAsk,
		),
		AmountOfPeersHoldingAQueryWord: amountOfPeersAcross(
			answeredAsksWithAHeldDocument(round.answeredAsks),
			peerOfAnsweredMatchedAndHeldDocumentsAsk,
		),
		AmountOfAnchorDocuments: len(round.anchor().documentsHeld()),
		AmountOfMatchedDocumentsAcrossAnswers: amountOfMatchedDocumentsAcrossAnswers(
			round.answeredAsks,
		),
		AmountOfMatchedDocumentsCountedByAPeer: amountOfMatchedDocumentsCountedByAPeer(
			round.answeredAsks,
		),
		AmountOfDocumentsHeldInEachAnswer: amountOfDocumentsHeldInEachAnswer(round.answeredAsks),
	}
}

func amountOfQueryWordsHeldByNoPeerAmong(queryWords []answeredQueryWord) int {
	amount := 0
	for _, queryWord := range queryWords {
		if len(queryWord.documentsHeld()) > 0 {
			continue
		}
		amount++
	}

	return amount
}

func amountOfFullyListedQueryWordsAmong(queryWords []answeredQueryWord) int {
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
