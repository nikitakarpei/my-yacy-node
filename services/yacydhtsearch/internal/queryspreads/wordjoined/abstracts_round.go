package wordjoined

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type abstractsRound struct {
	queryWords                     []yacymodel.Hash
	peersAsked                     map[yacymodel.Hash]struct{}
	answeredAsks                   []peerasks.AnsweredSearchDocumentsAsk
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas
	compoundWords                  []compoundWordAcrossReplicas
	amountOfPeersPerDocument       map[yacymodel.URLHash]int
}

func peersAskedIn(asks []peerasks.SearchDocumentsAsk) map[yacymodel.Hash]struct{} {
	peersAsked := make(map[yacymodel.Hash]struct{}, len(asks))
	for _, ask := range asks {
		peersAsked[ask.Peer.Hash] = struct{}{}
	}

	return peersAsked
}

func (round abstractsRound) leadingQueryWord() queryWordAcrossReplicas {
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		if queryWord.isFullyListed() {
			return queryWord
		}
	}

	return round.queryWordsFewestDocumentsFirst[0]
}

func (round abstractsRound) queryWordsBesideTheLeadingQueryWord() []queryWordAcrossReplicas {
	leadingQueryWord := round.leadingQueryWord().word

	return slices.DeleteFunc(
		slices.Clone(round.queryWordsFewestDocumentsFirst),
		func(queryWord queryWordAcrossReplicas) bool { return queryWord.word == leadingQueryWord },
	)
}

func (round abstractsRound) documentsOfTheLeadingQueryWordMostListedFirst() []yacymodel.URLHash {
	return round.documentsMostListedFirstAmong(round.leadingQueryWord().documentsListedByPeers())
}

func (round abstractsRound) documentsMostListedFirstAmong(
	documents distinctDocuments,
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

func (round abstractsRound) amountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(
		map[yacymodel.Hash]int, len(round.queryWordsFewestDocumentsFirst),
	)
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		amountOfDocumentsHeld, counted := queryWord.estimatedAmountOfDocumentsHeld().Get()
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
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
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

func (round abstractsRound) documentsListedByPeersPerQueryWord() documentsPerQueryWord {
	documentsListedByPeersPerQueryWord := make(
		documentsPerQueryWord,
		len(round.queryWordsFewestDocumentsFirst),
	)
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		documentsListedByPeersPerQueryWord[queryWord.word] = queryWord.documentsListedByPeers()
	}
	for _, compoundWord := range round.compoundWords {
		for _, word := range compoundWord.WordHashes {
			documentsListedByPeersPerQueryWord.add(word, compoundWord.documentsListedByPeers())
		}
	}

	return documentsListedByPeersPerQueryWord
}
