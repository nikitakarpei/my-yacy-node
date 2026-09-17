package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func matchedAndHeldDocumentsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord []peerchoice.PeersOfQueryWord,
	itemsCeiling int,
) []peerasks.MatchedAndHeldDocumentsAsk {
	return asksInTurnsAcrossTheWordsOf(
		matchedAndHeldDocumentsAsksPerQueryWordFor(query, chosenPeersPerQueryWord, itemsCeiling),
	)
}

func matchedAndHeldDocumentsAsksPerQueryWordFor(
	query searchquery.Query,
	chosenPeersPerQueryWord []peerchoice.PeersOfQueryWord,
	itemsCeiling int,
) [][]peerasks.MatchedAndHeldDocumentsAsk {
	asksPerQueryWord := make([][]peerasks.MatchedAndHeldDocumentsAsk, 0, len(query.TermHashes()))
	for index, queryWord := range query.TermHashes() {
		chosenPeers := chosenPeersPerQueryWord[index].Peers()
		asksOfQueryWord := make([]peerasks.MatchedAndHeldDocumentsAsk, 0, len(chosenPeers))
		for _, peer := range chosenPeers {
			asksOfQueryWord = append(asksOfQueryWord, peerasks.MatchedAndHeldDocumentsAsk{
				Peer:          peer,
				Word:          queryWord,
				ExcludedWords: query.ExclusionHashes(),
				Language:      query.Language,
				ItemsCeiling:  itemsCeiling,
			})
		}
		asksPerQueryWord = append(asksPerQueryWord, asksOfQueryWord)
	}

	return asksPerQueryWord
}

func asksInTurnsAcrossTheWordsOf(
	asksPerQueryWord [][]peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.MatchedAndHeldDocumentsAsk {
	var asksInTurns []peerasks.MatchedAndHeldDocumentsAsk
	for turn := range amountOfTurnsAcrossTheWordsOf(asksPerQueryWord) {
		for _, asksOfQueryWord := range asksPerQueryWord {
			if turn >= len(asksOfQueryWord) {
				continue
			}
			asksInTurns = append(asksInTurns, asksOfQueryWord[turn])
		}
	}

	return asksInTurns
}

func amountOfTurnsAcrossTheWordsOf(asksPerQueryWord [][]peerasks.MatchedAndHeldDocumentsAsk) int {
	amountOfTurns := 0
	for _, asksOfQueryWord := range asksPerQueryWord {
		amountOfTurns = max(amountOfTurns, len(asksOfQueryWord))
	}

	return amountOfTurns
}
