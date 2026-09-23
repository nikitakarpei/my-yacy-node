package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discoveryRound struct {
	queryWords                     []yacymodel.Hash
	peersAsked                     map[yacymodel.Hash]struct{}
	answeredAsks                   []peerasks.AnsweredSearchDocumentsAsk
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas
	compoundWords                  []compoundWordAcrossReplicas
	holdersPerDocument             holdersPerDocument
	time                           yacymodel.Optional[RoundTime]
}

func peersAskedIn(asks []peerasks.SearchDocumentsAsk) map[yacymodel.Hash]struct{} {
	peersAsked := make(map[yacymodel.Hash]struct{}, len(asks))
	for _, ask := range asks {
		peersAsked[ask.Peer.Hash] = struct{}{}
	}

	return peersAsked
}

func (round discoveryRound) leadingQueryWord() queryWordAcrossReplicas {
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		if queryWord.hasCompleteAbstracts() {
			return queryWord
		}
	}

	return round.queryWordsFewestDocumentsFirst[0]
}

func (round discoveryRound) queryWordsBesideTheLeadingQueryWord() []queryWordAcrossReplicas {
	leadingQueryWord := round.leadingQueryWord().word

	return slices.DeleteFunc(
		slices.Clone(round.queryWordsFewestDocumentsFirst),
		func(queryWord queryWordAcrossReplicas) bool { return queryWord.word == leadingQueryWord },
	)
}

func (round discoveryRound) amountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
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

func (round discoveryRound) documentsPerQueryWord() documentsPerQueryWord {
	documentsOfEachQueryWord := make(
		documentsPerQueryWord,
		len(round.queryWordsFewestDocumentsFirst),
	)
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		documentsOfEachQueryWord[queryWord.word] = queryWord.documents()
	}
	for _, compoundWord := range round.compoundWords {
		for _, word := range compoundWord.WordHashes {
			documentsOfEachQueryWord.add(word, compoundWord.documents())
		}
	}

	return documentsOfEachQueryWord
}
