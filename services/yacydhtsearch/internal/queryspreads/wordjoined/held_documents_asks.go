package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

func heldDocumentsAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerdirectory.AskablePeer,
	itemsCeiling int,
) []peerasks.HeldDocumentsAsk {
	return asksInTurnsAcrossTheWordsOf(
		heldDocumentsAsksPerQueryWordFor(query, chosenPeersPerQueryWord, itemsCeiling),
	)
}

func heldDocumentsAsksPerQueryWordFor(
	query searchquery.Query,
	chosenPeersPerQueryWord [][]peerdirectory.AskablePeer,
	itemsCeiling int,
) [][]peerasks.HeldDocumentsAsk {
	asksPerQueryWord := make([][]peerasks.HeldDocumentsAsk, 0, len(query.TermHashes()))
	for index, queryWord := range query.TermHashes() {
		asksOfQueryWord := make(
			[]peerasks.HeldDocumentsAsk, 0, len(chosenPeersPerQueryWord[index]),
		)
		for _, peer := range chosenPeersPerQueryWord[index] {
			asksOfQueryWord = append(asksOfQueryWord, peerasks.HeldDocumentsAsk{
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
	asksPerQueryWord [][]peerasks.HeldDocumentsAsk,
) []peerasks.HeldDocumentsAsk {
	var asksInTurns []peerasks.HeldDocumentsAsk
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

func amountOfTurnsAcrossTheWordsOf(asksPerQueryWord [][]peerasks.HeldDocumentsAsk) int {
	turns := 0
	for _, asksOfQueryWord := range asksPerQueryWord {
		turns = max(turns, len(asksOfQueryWord))
	}

	return turns
}
