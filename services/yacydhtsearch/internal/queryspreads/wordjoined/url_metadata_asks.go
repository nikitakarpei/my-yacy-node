package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	documentsWithoutMetadataMostListedFirst []yacymodel.URLHash,
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	urlMetadataAskDocumentsCeiling int,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	documentsListedByEachPeer := documentsListedByEachPeerAmong(
		documentsWithoutMetadataMostListedFirst,
		answeredMatchedAndHeldDocumentsAsks,
		urlMetadataAskDocumentsCeiling,
	)
	coveringPeers := peersCoveringMostDocuments(
		documentsListedByEachPeer, amountOfPeersHoldingOneWord,
	)

	asks := make([]peerasks.URLMetadataAsk, 0, len(coveringPeers))
	for _, documentsListedByOnePeer := range coveringPeers {
		asks = append(asks, peerasks.URLMetadataAsk{
			Peer:      documentsListedByOnePeer.peer,
			Documents: documentsListedByOnePeer.documents,
		})
	}

	return asks
}

type documentsListedByPeer struct {
	peer      peerdirectory.AskablePeer
	documents []yacymodel.URLHash
}

func documentsListedByEachPeerAmong(
	documentsMostListedFirst []yacymodel.URLHash,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	urlMetadataAskDocumentsCeiling int,
) []documentsListedByPeer {
	peersInListingOrder, documentsHeldByEachPeer := documentsHeldByEachPeerOf(answeredAsks)
	documentsListedByEachPeer := make([]documentsListedByPeer, 0, len(peersInListingOrder))
	for _, peer := range peersInListingOrder {
		listedDocuments := documentsMostListedFirstHeldIn(
			documentsMostListedFirst,
			documentsHeldByEachPeer[peer.Hash],
			urlMetadataAskDocumentsCeiling,
		)
		if len(listedDocuments) == 0 {
			continue
		}
		documentsListedByEachPeer = append(documentsListedByEachPeer, documentsListedByPeer{
			peer:      peer,
			documents: listedDocuments,
		})
	}

	return documentsListedByEachPeer
}

func documentsHeldByEachPeerOf(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) ([]peerdirectory.AskablePeer, map[yacymodel.Hash]distinctDocuments) {
	peersInListingOrder := make([]peerdirectory.AskablePeer, 0, len(answeredAsks))
	documentsHeldByEachPeer := make(map[yacymodel.Hash]distinctDocuments, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		documentsHeldByThePeer, known := documentsHeldByEachPeer[answeredAsk.Ask.Peer.Hash]
		if !known {
			documentsHeldByThePeer = distinctDocuments{}
			documentsHeldByEachPeer[answeredAsk.Ask.Peer.Hash] = documentsHeldByThePeer
			peersInListingOrder = append(peersInListingOrder, answeredAsk.Ask.Peer)
		}
		for _, document := range answeredAsk.DocumentsListedForTheWord {
			documentsHeldByThePeer.add(document)
		}
	}

	return peersInListingOrder, documentsHeldByEachPeer
}

func documentsMostListedFirstHeldIn(
	documentsMostListedFirst []yacymodel.URLHash,
	documentsHeldByThePeer distinctDocuments,
	urlMetadataAskDocumentsCeiling int,
) []yacymodel.URLHash {
	heldDocuments := make([]yacymodel.URLHash, 0, min(
		len(documentsHeldByThePeer), urlMetadataAskDocumentsCeiling,
	))
	for _, document := range documentsMostListedFirst {
		if len(heldDocuments) == urlMetadataAskDocumentsCeiling {
			break
		}
		if !documentsHeldByThePeer.contains(document) {
			continue
		}
		heldDocuments = append(heldDocuments, document)
	}

	return heldDocuments
}

func peersCoveringMostDocuments(
	documentsListedByEachPeer []documentsListedByPeer,
	amountOfPeersHoldingOneWord int,
) []documentsListedByPeer {
	if len(documentsListedByEachPeer) <= amountOfPeersHoldingOneWord {
		return documentsListedByEachPeer
	}

	coveringPeers := make([]documentsListedByPeer, 0, amountOfPeersHoldingOneWord)
	coveredDocuments := distinctDocuments{}
	takenPeers := make([]bool, len(documentsListedByEachPeer))
	for len(coveringPeers) < amountOfPeersHoldingOneWord {
		mostCoveringPeer := mostCoveringPeerAmong(
			documentsListedByEachPeer, takenPeers, coveredDocuments,
		)
		if mostCoveringPeer.amountOfUncoveredDocuments == 0 {
			break
		}
		takenPeers[mostCoveringPeer.place] = true
		for _, document := range documentsListedByEachPeer[mostCoveringPeer.place].documents {
			coveredDocuments.add(document)
		}
		coveringPeers = append(coveringPeers, documentsListedByEachPeer[mostCoveringPeer.place])
	}

	return coveringPeers
}

type mostCoveringPeer struct {
	place                      int
	amountOfUncoveredDocuments int
}

func mostCoveringPeerAmong(
	documentsListedByEachPeer []documentsListedByPeer,
	takenPeers []bool,
	coveredDocuments distinctDocuments,
) mostCoveringPeer {
	mostCoveringPeer := mostCoveringPeer{}
	for place, documentsListedByOnePeer := range documentsListedByEachPeer {
		if takenPeers[place] {
			continue
		}
		amountOfUncoveredDocuments := amountOfDocumentsNotCovered(
			documentsListedByOnePeer.documents, coveredDocuments,
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
	coveredDocuments distinctDocuments,
) int {
	amount := 0
	for _, document := range documents {
		if coveredDocuments.contains(document) {
			continue
		}
		amount++
	}

	return amount
}
