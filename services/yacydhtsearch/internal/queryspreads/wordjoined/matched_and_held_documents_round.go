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
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas
	compoundWords                  []compoundWordAcrossReplicas
	amountOfPeersPerDocument       map[yacymodel.URLHash]int
}

func (round matchedAndHeldDocumentsRound) leadingQueryWord() queryWordAcrossReplicas {
	return round.queryWordsFewestDocumentsFirst[round.placeOfTheLeadingQueryWord()]
}

func (round matchedAndHeldDocumentsRound) placeOfTheLeadingQueryWord() int {
	return max(
		0,
		slices.IndexFunc(
			round.queryWordsFewestDocumentsFirst,
			queryWordAcrossReplicas.isFullyListed,
		),
	)
}

func (round matchedAndHeldDocumentsRound) queryWordsBesideTheLeadingQueryWord() []queryWordAcrossReplicas {
	placeOfTheLeadingQueryWord := round.placeOfTheLeadingQueryWord()

	return slices.Concat(
		round.queryWordsFewestDocumentsFirst[:placeOfTheLeadingQueryWord],
		round.queryWordsFewestDocumentsFirst[placeOfTheLeadingQueryWord+1:],
	)
}

func (round matchedAndHeldDocumentsRound) partlyListedQueryWordsBesideTheLeadingQueryWord() []queryWordAcrossReplicas {
	var partlyListedQueryWords []queryWordAcrossReplicas
	for _, queryWord := range round.queryWordsBesideTheLeadingQueryWord() {
		if queryWord.isFullyListed() {
			continue
		}
		partlyListedQueryWords = append(partlyListedQueryWords, queryWord)
	}

	return partlyListedQueryWords
}

func (round matchedAndHeldDocumentsRound) replicasThatMayCrossCheck() []queryWordOnReplica {
	var replicas []queryWordOnReplica
	for _, queryWord := range round.partlyListedQueryWordsBesideTheLeadingQueryWord() {
		for _, replica := range queryWord.replicasThatDidNotListAllTheyHold() {
			if slices.ContainsFunc(replicas, func(known queryWordOnReplica) bool {
				return known.peer.Hash == replica.peer.Hash
			}) {
				continue
			}
			replicas = append(replicas, replica)
		}
	}

	return replicas
}

func (round matchedAndHeldDocumentsRound) documentsOfTheLeadingQueryWordMostListedFirst() []yacymodel.URLHash {
	return round.documentsMostListedFirstAmong(round.leadingQueryWord().documentsListedByPeers())
}

func (round matchedAndHeldDocumentsRound) amountOfCrossCheckCandidates() int {
	documentsOfTheLeadingQueryWord := round.documentsOfTheLeadingQueryWordMostListedFirst()
	amount := 0
	for _, queryWord := range round.partlyListedQueryWordsBesideTheLeadingQueryWord() {
		amount += len(queryWord.crossCheckCandidatesAmong(documentsOfTheLeadingQueryWord))
	}

	return amount
}

func (round matchedAndHeldDocumentsRound) documentsMostListedFirstAmong(
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

func (round matchedAndHeldDocumentsRound) amountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
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

func (round matchedAndHeldDocumentsRound) documentsListedByPeersPerQueryWord() documentsPerQueryWord {
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
