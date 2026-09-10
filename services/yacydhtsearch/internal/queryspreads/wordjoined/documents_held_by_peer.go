package wordjoined

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type documentsHeldByPeer struct {
	peer      peerdirectory.AskablePeer
	documents []yacymodel.URLHash
}

func joinedDocumentsOf(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	queryWords []yacymodel.Hash,
) map[yacymodel.URLHash]struct{} {
	joinedDocuments := map[yacymodel.URLHash]struct{}{}
	for document, amountOfWords := range amountOfQueryWordsPerDocument(answeredAsks) {
		if amountOfWords != len(queryWords) {
			continue
		}
		joinedDocuments[document] = struct{}{}
	}

	return joinedDocuments
}

type queryWordOfDocument struct {
	document  yacymodel.URLHash
	queryWord yacymodel.Hash
}

func amountOfQueryWordsPerDocument(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) map[yacymodel.URLHash]int {
	countedWords := map[queryWordOfDocument]struct{}{}
	amountOfWords := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, document := range answeredAsk.Documents {
			ofDocument := queryWordOfDocument{document: document, queryWord: answeredAsk.Ask.Word}
			if _, counted := countedWords[ofDocument]; counted {
				continue
			}
			countedWords[ofDocument] = struct{}{}
			amountOfWords[document]++
		}
	}

	return amountOfWords
}

func mostHeldDocumentsAmong(
	joinedDocuments map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	documentsToAskMetadataForCeiling int,
) map[yacymodel.URLHash]struct{} {
	if len(joinedDocuments) <= documentsToAskMetadataForCeiling {
		return joinedDocuments
	}

	amountOfPeers := amountOfPeersPerDocument(answeredAsks)
	mostHeldFirst := slices.SortedFunc(
		maps.Keys(joinedDocuments),
		func(first, second yacymodel.URLHash) int {
			if amountOfPeers[first] != amountOfPeers[second] {
				return cmp.Compare(amountOfPeers[second], amountOfPeers[first])
			}

			return strings.Compare(first.String(), second.String())
		},
	)

	mostHeld := make(map[yacymodel.URLHash]struct{}, documentsToAskMetadataForCeiling)
	for _, document := range mostHeldFirst[:documentsToAskMetadataForCeiling] {
		mostHeld[document] = struct{}{}
	}

	return mostHeld
}

type peerOfDocument struct {
	document yacymodel.URLHash
	peer     yacymodel.Hash
}

func amountOfPeersPerDocument(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) map[yacymodel.URLHash]int {
	countedPeers := map[peerOfDocument]struct{}{}
	amountOfPeers := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, document := range answeredAsk.Documents {
			ofDocument := peerOfDocument{document: document, peer: answeredAsk.Ask.Peer.Hash}
			if _, counted := countedPeers[ofDocument]; counted {
				continue
			}
			countedPeers[ofDocument] = struct{}{}
			amountOfPeers[document]++
		}
	}

	return amountOfPeers
}

func documentsHeldPerPeerAmong(
	documentsToAskMetadataFor map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []documentsHeldByPeer {
	heldPerPeer := make([]documentsHeldByPeer, 0, len(answeredAsks))
	indexOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		documents := documentsAmong(answeredAsk.Documents, documentsToAskMetadataFor)
		if len(documents) == 0 {
			continue
		}
		index, holds := indexOfPeer[answeredAsk.Ask.Peer.Hash]
		if !holds {
			index = len(heldPerPeer)
			indexOfPeer[answeredAsk.Ask.Peer.Hash] = index
			heldPerPeer = append(heldPerPeer, documentsHeldByPeer{peer: answeredAsk.Ask.Peer})
		}
		heldPerPeer[index].documents = append(heldPerPeer[index].documents, documents...)
	}
	for index, heldByPeer := range heldPerPeer {
		heldPerPeer[index].documents = documentsWithoutRepeats(heldByPeer.documents)
	}

	return heldPerPeer
}

func documentsAmong(
	documents []yacymodel.URLHash,
	documentsToAskMetadataFor map[yacymodel.URLHash]struct{},
) []yacymodel.URLHash {
	kept := make([]yacymodel.URLHash, 0, len(documents))
	for _, document := range documents {
		if _, toAskMetadataFor := documentsToAskMetadataFor[document]; !toAskMetadataFor {
			continue
		}
		kept = append(kept, document)
	}

	return kept
}

func documentsWithoutRepeats(documents []yacymodel.URLHash) []yacymodel.URLHash {
	kept := make([]yacymodel.URLHash, 0, len(documents))
	seenDocuments := make(map[yacymodel.URLHash]struct{}, len(documents))
	for _, document := range documents {
		if _, seen := seenDocuments[document]; seen {
			continue
		}
		seenDocuments[document] = struct{}{}
		kept = append(kept, document)
	}

	return kept
}

func peersCoveringMostDocuments(
	heldPerPeer []documentsHeldByPeer,
	peersCeiling int,
) []documentsHeldByPeer {
	if len(heldPerPeer) <= peersCeiling {
		return heldPerPeer
	}

	covering := make([]documentsHeldByPeer, 0, peersCeiling)
	coveredDocuments := map[yacymodel.URLHash]struct{}{}
	takenPeers := make([]bool, len(heldPerPeer))
	for len(covering) < peersCeiling {
		mostCovering := mostCoveringPeerAmong(heldPerPeer, takenPeers, coveredDocuments)
		if mostCovering.uncoveredDocuments == 0 {
			break
		}
		takenPeers[mostCovering.peer] = true
		for _, document := range heldPerPeer[mostCovering.peer].documents {
			coveredDocuments[document] = struct{}{}
		}
		covering = append(covering, heldPerPeer[mostCovering.peer])
	}

	return covering
}

type mostCoveringPeer struct {
	peer               int
	uncoveredDocuments int
}

func mostCoveringPeerAmong(
	heldPerPeer []documentsHeldByPeer,
	takenPeers []bool,
	coveredDocuments map[yacymodel.URLHash]struct{},
) mostCoveringPeer {
	mostCovering := mostCoveringPeer{}
	for index, heldByPeer := range heldPerPeer {
		if takenPeers[index] {
			continue
		}
		uncovered := amountOfDocumentsNotCovered(heldByPeer.documents, coveredDocuments)
		if uncovered > mostCovering.uncoveredDocuments {
			mostCovering = mostCoveringPeer{peer: index, uncoveredDocuments: uncovered}
		}
	}

	return mostCovering
}

func amountOfDocumentsNotCovered(
	documents []yacymodel.URLHash,
	coveredDocuments map[yacymodel.URLHash]struct{},
) int {
	amount := 0
	for _, document := range documents {
		if _, covered := coveredDocuments[document]; covered {
			continue
		}
		amount++
	}

	return amount
}
