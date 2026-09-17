package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	documentsWithoutMetadataMostHeldFirst []yacymodel.URLHash,
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	metadataDocumentsCeiling int,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	mostHeldDocuments := mostHeldDocumentsFrom(
		documentsWithoutMetadataMostHeldFirst, metadataDocumentsCeiling,
	)
	documentsHeldByEachPeer := documentsHeldByEachPeerAmong(
		mostHeldDocuments, answeredMatchedAndHeldDocumentsAsks,
	)
	coveringPeers := peersCoveringMostDocuments(
		documentsHeldByEachPeer, amountOfPeersHoldingOneWord,
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

func mostHeldDocumentsFrom(
	documentsMostHeldFirst []yacymodel.URLHash,
	metadataDocumentsCeiling int,
) map[yacymodel.URLHash]struct{} {
	mostHeldDocuments := make(map[yacymodel.URLHash]struct{}, metadataDocumentsCeiling)
	for _, document := range documentsMostHeldFirst[:min(
		len(documentsMostHeldFirst), metadataDocumentsCeiling,
	)] {
		mostHeldDocuments[document] = struct{}{}
	}

	return mostHeldDocuments
}

type documentsHeldByPeer struct {
	peer      peerdirectory.AskablePeer
	documents []yacymodel.URLHash
}

func documentsHeldByEachPeerAmong(
	documents map[yacymodel.URLHash]struct{},
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
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
	amountOfPeersHoldingOneWord int,
) []documentsHeldByPeer {
	if len(documentsHeldByEachPeer) <= amountOfPeersHoldingOneWord {
		return documentsHeldByEachPeer
	}

	coveringPeers := make([]documentsHeldByPeer, 0, amountOfPeersHoldingOneWord)
	coveredDocuments := map[yacymodel.URLHash]struct{}{}
	takenPeers := make([]bool, len(documentsHeldByEachPeer))
	for len(coveringPeers) < amountOfPeersHoldingOneWord {
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
