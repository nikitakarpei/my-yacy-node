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

func urlMetadataAsksFor(
	documentsWithoutMetadata map[yacymodel.URLHash]struct{},
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	metadataDocumentsCeiling int,
	peersCeiling int,
) []peerasks.URLMetadataAsk {
	mostHeldDocuments := mostHeldDocumentsAmong(
		documentsWithoutMetadata, answeredHeldDocumentsAsks, metadataDocumentsCeiling,
	)
	documentsHeldByEachPeer := documentsHeldByEachPeerAmong(
		mostHeldDocuments, answeredHeldDocumentsAsks,
	)
	coveringPeers := peersCoveringMostDocuments(documentsHeldByEachPeer, peersCeiling)

	asks := make([]peerasks.URLMetadataAsk, 0, len(coveringPeers))
	for _, heldByPeer := range coveringPeers {
		asks = append(asks, peerasks.URLMetadataAsk{
			Peer:      heldByPeer.peer,
			Documents: heldByPeer.documents,
		})
	}

	return asks
}

func mostHeldDocumentsAmong(
	documents map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	metadataDocumentsCeiling int,
) map[yacymodel.URLHash]struct{} {
	if len(documents) <= metadataDocumentsCeiling {
		return documents
	}

	amountOfPeers := amountOfPeersPerDocument(answeredAsks)
	mostHeldFirst := slices.SortedFunc(
		maps.Keys(documents),
		func(first, second yacymodel.URLHash) int {
			if amountOfPeers[first] != amountOfPeers[second] {
				return cmp.Compare(amountOfPeers[second], amountOfPeers[first])
			}

			return strings.Compare(first.String(), second.String())
		},
	)

	mostHeld := make(map[yacymodel.URLHash]struct{}, metadataDocumentsCeiling)
	for _, document := range mostHeldFirst[:metadataDocumentsCeiling] {
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
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
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

type documentsHeldByPeer struct {
	peer      peerdirectory.AskablePeer
	documents []yacymodel.URLHash
}

func documentsHeldByEachPeerAmong(
	documents map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []documentsHeldByPeer {
	heldByEachPeer := make([]documentsHeldByPeer, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		heldDocuments := documentsHeldAmong(answeredAsk.DocumentsHeldForTheWord, documents)
		if len(heldDocuments) == 0 {
			continue
		}
		place, holds := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !holds {
			place = len(heldByEachPeer)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			heldByEachPeer = append(heldByEachPeer, documentsHeldByPeer{peer: answeredAsk.Ask.Peer})
		}
		heldByEachPeer[place].documents = append(heldByEachPeer[place].documents, heldDocuments...)
	}
	for place, heldByPeer := range heldByEachPeer {
		heldByEachPeer[place].documents = documentsWithoutRepeats(heldByPeer.documents)
	}

	return heldByEachPeer
}

func documentsHeldAmong(
	heldDocuments []yacymodel.URLHash,
	documents map[yacymodel.URLHash]struct{},
) []yacymodel.URLHash {
	kept := make([]yacymodel.URLHash, 0, len(heldDocuments))
	for _, document := range heldDocuments {
		if _, among := documents[document]; !among {
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
	documentsHeldByEachPeer []documentsHeldByPeer,
	peersCeiling int,
) []documentsHeldByPeer {
	if len(documentsHeldByEachPeer) <= peersCeiling {
		return documentsHeldByEachPeer
	}

	covering := make([]documentsHeldByPeer, 0, peersCeiling)
	coveredDocuments := map[yacymodel.URLHash]struct{}{}
	takenPeers := make([]bool, len(documentsHeldByEachPeer))
	for len(covering) < peersCeiling {
		mostCovering := mostCoveringPeerAmong(
			documentsHeldByEachPeer, takenPeers, coveredDocuments,
		)
		if mostCovering.uncoveredDocuments == 0 {
			break
		}
		takenPeers[mostCovering.place] = true
		for _, document := range documentsHeldByEachPeer[mostCovering.place].documents {
			coveredDocuments[document] = struct{}{}
		}
		covering = append(covering, documentsHeldByEachPeer[mostCovering.place])
	}

	return covering
}

type mostCoveringPeer struct {
	place              int
	uncoveredDocuments int
}

func mostCoveringPeerAmong(
	documentsHeldByEachPeer []documentsHeldByPeer,
	takenPeers []bool,
	coveredDocuments map[yacymodel.URLHash]struct{},
) mostCoveringPeer {
	mostCovering := mostCoveringPeer{}
	for place, heldByPeer := range documentsHeldByEachPeer {
		if takenPeers[place] {
			continue
		}
		uncovered := amountOfDocumentsNotCovered(heldByPeer.documents, coveredDocuments)
		if uncovered > mostCovering.uncoveredDocuments {
			mostCovering = mostCoveringPeer{place: place, uncoveredDocuments: uncovered}
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
