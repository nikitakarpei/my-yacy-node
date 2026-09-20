package wordjoined

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type matchedAndHeldDocumentsRound struct {
	queryWords                     []yacymodel.Hash
	asks                           []peerasks.MatchedAndHeldDocumentsAsk
	answeredAsks                   []peerasks.AnsweredMatchedAndHeldDocumentsAsk
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas
	amountOfPeersPerDocument       map[yacymodel.URLHash]int
	reliabilityOfEachPeer          map[yacymodel.Hash]float64
}

func (round matchedAndHeldDocumentsRound) placesGivenByPeersPerDocument() map[yacymodel.URLHash][]queryanswers.PlaceGivenByPeer {
	placesGivenByPeersPerDocument := map[yacymodel.URLHash][]queryanswers.PlaceGivenByPeer{}
	for _, answeredAsk := range round.answeredAsks {
		for place, document := range answeredAsk.DocumentsListedForTheWord {
			placesGivenByPeersPerDocument[document] = append(
				placesGivenByPeersPerDocument[document],
				queryanswers.PlaceGivenByPeer{
					Peer:                 answeredAsk.Ask.Peer.Hash,
					ReliabilityOfThePeer: round.reliabilityOfEachPeer[answeredAsk.Ask.Peer.Hash],
					Place:                place,
				},
			)
		}
	}

	return placesGivenByPeersPerDocument
}

func (round matchedAndHeldDocumentsRound) leadingQueryWord() queryWordAcrossReplicas {
	return round.queryWordsFewestDocumentsFirst[round.placeOfTheLeadingQueryWord()]
}

func (round matchedAndHeldDocumentsRound) queryWordsBesideTheLeadingQueryWord() []queryWordAcrossReplicas {
	placeOfTheLeadingQueryWord := round.placeOfTheLeadingQueryWord()

	return slices.Concat(
		round.queryWordsFewestDocumentsFirst[:placeOfTheLeadingQueryWord],
		round.queryWordsFewestDocumentsFirst[placeOfTheLeadingQueryWord+1:],
	)
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

	return documentsListedByPeersPerQueryWord
}
