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
	amountOfPeersAskedPerWord int,
) []peerasks.URLMetadataAsk {
	mostHeldDocuments := mostHeldDocumentsAmong(
		documentsWithoutMetadata, answeredHeldDocumentsAsks, metadataDocumentsCeiling,
	)
	documentsHeldByEachPeer := documentsHeldByEachPeerAmong(
		mostHeldDocuments, answeredHeldDocumentsAsks,
	)
	coveringPeers := peersCoveringMostDocuments(
		documentsHeldByEachPeer, amountOfPeersAskedPerWord,
	)

	asks := make([]peerasks.URLMetadataAsk, 0, len(coveringPeers))
	for _, documentsHeldByOnePeer := range coveringPeers {
		asks = append(asks, peerasks.URLMetadataAsk{
			Peer:      documentsHeldByOnePeer.peer,
			Documents: documentsHeldByOnePeer.documents,
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

	amountOfPeersPerDocument := amountOfPeersPerDocument(answeredAsks)
	documentsMostHeldFirst := slices.SortedFunc(
		maps.Keys(documents),
		func(first, second yacymodel.URLHash) int {
			if amountOfPeersPerDocument[first] != amountOfPeersPerDocument[second] {
				return cmp.Compare(
					amountOfPeersPerDocument[second], amountOfPeersPerDocument[first],
				)
			}

			return strings.Compare(first.String(), second.String())
		},
	)

	mostHeldDocuments := make(map[yacymodel.URLHash]struct{}, metadataDocumentsCeiling)
	for _, document := range documentsMostHeldFirst[:metadataDocumentsCeiling] {
		mostHeldDocuments[document] = struct{}{}
	}

	return mostHeldDocuments
}

type peerOfDocument struct {
	document yacymodel.URLHash
	peer     yacymodel.Hash
}

func amountOfPeersPerDocument(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) map[yacymodel.URLHash]int {
	countedPeers := map[peerOfDocument]struct{}{}
	amountOfPeersPerDocument := map[yacymodel.URLHash]int{}
	for _, answeredAsk := range answeredAsks {
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
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

type documentsHeldByPeer struct {
	peer      peerdirectory.AskablePeer
	documents []yacymodel.URLHash
}

func documentsHeldByEachPeerAmong(
	documents map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) []documentsHeldByPeer {
	documentsHeldByEachPeer := make([]documentsHeldByPeer, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		heldDocuments := documentsHeldAmong(documents, answeredAsk.DocumentsHeldForTheWord)
		if len(heldDocuments) == 0 {
			continue
		}
		place, holds := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !holds {
			place = len(documentsHeldByEachPeer)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			documentsHeldByEachPeer = append(
				documentsHeldByEachPeer, documentsHeldByPeer{peer: answeredAsk.Ask.Peer},
			)
		}
		documentsHeldByEachPeer[place].documents = append(
			documentsHeldByEachPeer[place].documents, heldDocuments...,
		)
	}
	for place, documentsHeldByOnePeer := range documentsHeldByEachPeer {
		documentsHeldByEachPeer[place].documents = documentsWithoutRepeats(
			documentsHeldByOnePeer.documents,
		)
	}

	return documentsHeldByEachPeer
}

func documentsHeldAmong(
	documents map[yacymodel.URLHash]struct{},
	heldDocuments []yacymodel.URLHash,
) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(heldDocuments))
	for _, document := range heldDocuments {
		if _, among := documents[document]; !among {
			continue
		}
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
}

func documentsWithoutRepeats(documents []yacymodel.URLHash) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(documents))
	seenDocuments := make(map[yacymodel.URLHash]struct{}, len(documents))
	for _, document := range documents {
		if _, seen := seenDocuments[document]; seen {
			continue
		}
		seenDocuments[document] = struct{}{}
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
}

func peersCoveringMostDocuments(
	documentsHeldByEachPeer []documentsHeldByPeer,
	amountOfPeersAskedPerWord int,
) []documentsHeldByPeer {
	if len(documentsHeldByEachPeer) <= amountOfPeersAskedPerWord {
		return documentsHeldByEachPeer
	}

	coveringPeers := make([]documentsHeldByPeer, 0, amountOfPeersAskedPerWord)
	coveredDocuments := map[yacymodel.URLHash]struct{}{}
	takenPeers := make([]bool, len(documentsHeldByEachPeer))
	for len(coveringPeers) < amountOfPeersAskedPerWord {
		mostCoveringPeer := mostCoveringPeerAmong(
			documentsHeldByEachPeer, takenPeers, coveredDocuments,
		)
		if mostCoveringPeer.amountOfUncoveredDocuments == 0 {
			break
		}
		takenPeers[mostCoveringPeer.place] = true
		for _, document := range documentsHeldByEachPeer[mostCoveringPeer.place].documents {
			coveredDocuments[document] = struct{}{}
		}
		coveringPeers = append(coveringPeers, documentsHeldByEachPeer[mostCoveringPeer.place])
	}

	return coveringPeers
}

type mostCoveringPeer struct {
	place                      int
	amountOfUncoveredDocuments int
}

func mostCoveringPeerAmong(
	documentsHeldByEachPeer []documentsHeldByPeer,
	takenPeers []bool,
	coveredDocuments map[yacymodel.URLHash]struct{},
) mostCoveringPeer {
	mostCoveringPeer := mostCoveringPeer{}
	for place, documentsHeldByOnePeer := range documentsHeldByEachPeer {
		if takenPeers[place] {
			continue
		}
		amountOfUncoveredDocuments := amountOfDocumentsNotCovered(
			documentsHeldByOnePeer.documents, coveredDocuments,
		)
		if amountOfUncoveredDocuments > mostCoveringPeer.amountOfUncoveredDocuments {
			mostCoveringPeer.place = place
			mostCoveringPeer.amountOfUncoveredDocuments = amountOfUncoveredDocuments
		}
	}

	return mostCoveringPeer
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
