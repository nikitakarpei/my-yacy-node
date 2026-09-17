package wordjoined

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type matchedAndHeldDocumentsRound struct {
	queryWords                     []yacymodel.Hash
	asks                           []peerasks.MatchedAndHeldDocumentsAsk
	answeredAsks                   []peerasks.AnsweredMatchedAndHeldDocumentsAsk
	queryWordsFewestDocumentsFirst []answeredQueryWord
	amountOfPeersPerDocument       map[yacymodel.URLHash]int
}

func (round matchedAndHeldDocumentsRound) leadingQueryWord() answeredQueryWord {
	return round.queryWordsFewestDocumentsFirst[round.placeOfTheLeadingQueryWord()]
}

func (round matchedAndHeldDocumentsRound) queryWordsBesideTheLeadingQueryWord() []answeredQueryWord {
	placeOfTheLeadingQueryWord := round.placeOfTheLeadingQueryWord()

	return slices.Concat(
		round.queryWordsFewestDocumentsFirst[:placeOfTheLeadingQueryWord],
		round.queryWordsFewestDocumentsFirst[placeOfTheLeadingQueryWord+1:],
	)
}

func (round matchedAndHeldDocumentsRound) placeOfTheLeadingQueryWord() int {
	return max(
		0,
		slices.IndexFunc(round.queryWordsFewestDocumentsFirst, answeredQueryWord.isFullyListed),
	)
}

func (round matchedAndHeldDocumentsRound) documentsMostListedFirstAmong(
	documents map[yacymodel.URLHash]struct{},
) []yacymodel.URLHash {
	return slices.SortedFunc(
		maps.Keys(documents),
		func(first, second yacymodel.URLHash) int {
			if round.amountOfPeersPerDocument[first] != round.amountOfPeersPerDocument[second] {
				return cmp.Compare(
					round.amountOfPeersPerDocument[second], round.amountOfPeersPerDocument[first],
				)
			}

			return strings.Compare(first.String(), second.String())
		},
	)
}

func (round matchedAndHeldDocumentsRound) amountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(
		map[yacymodel.Hash]int, len(round.queryWordsFewestDocumentsFirst),
	)
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		amountOfDocumentsHeld, counted := queryWord.amountOfDocumentsHeld()
		if !counted {
			continue
		}
		amountOfDocumentsHeldPerQueryWord[queryWord.word] = amountOfDocumentsHeld
	}

	return amountOfDocumentsHeldPerQueryWord
}

type peerOfDocument struct {
	document yacymodel.URLHash
	peer     yacymodel.Hash
}

func amountOfPeersPerDocumentOf(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) map[yacymodel.URLHash]int {
	countedPeers := map[peerOfDocument]struct{}{}
	amountOfPeersPerDocument := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, document := range answeredAsk.DocumentsListedForTheWord {
			peerOfDocument := peerOfDocument{document: document, peer: answeredAsk.Ask.Peer.Hash}
			if _, counted := countedPeers[peerOfDocument]; counted {
				continue
			}
			countedPeers[peerOfDocument] = struct{}{}
			amountOfPeersPerDocument[document]++
		}
	}

	return amountOfPeersPerDocument
}
