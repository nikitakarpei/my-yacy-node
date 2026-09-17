package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	documentsWithoutMetadataMostListedFirst []yacymodel.URLHash,
	answeredMatchedAndHeldDocumentsAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	metadataDocumentsCeiling int,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	mostListedDocuments := mostListedDocumentsFrom(
		documentsWithoutMetadataMostListedFirst, metadataDocumentsCeiling,
	)
	documentsListedByEachPeer := documentsListedByEachPeerAmong(
		mostListedDocuments, answeredMatchedAndHeldDocumentsAsks,
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

func mostListedDocumentsFrom(
	documentsMostListedFirst []yacymodel.URLHash,
	metadataDocumentsCeiling int,
) distinctDocuments {
	mostListedDocuments := make(distinctDocuments, metadataDocumentsCeiling)
	for _, document := range documentsMostListedFirst[:min(
		len(documentsMostListedFirst), metadataDocumentsCeiling,
	)] {
		mostListedDocuments.add(document)
	}

	return mostListedDocuments
}

type documentsListedByPeer struct {
	peer      peerdirectory.AskablePeer
	documents []yacymodel.URLHash
}

func documentsListedByEachPeerAmong(
	documents distinctDocuments,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) []documentsListedByPeer {
	documentsListedByEachPeer := make([]documentsListedByPeer, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		listedDocuments := listedDocumentsAmong(documents, answeredAsk.DocumentsListedForTheWord)
		if len(listedDocuments) == 0 {
			continue
		}
		place, placed := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !placed {
			place = len(documentsListedByEachPeer)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			documentsListedByEachPeer = append(
				documentsListedByEachPeer, documentsListedByPeer{peer: answeredAsk.Ask.Peer},
			)
		}
		documentsListedByEachPeer[place].documents = append(
			documentsListedByEachPeer[place].documents, listedDocuments...,
		)
	}
	for place, documentsListedByOnePeer := range documentsListedByEachPeer {
		documentsListedByEachPeer[place].documents = documentsWithoutRepeats(
			documentsListedByOnePeer.documents,
		)
	}

	return documentsListedByEachPeer
}

func listedDocumentsAmong(
	documents distinctDocuments,
	listedDocuments []yacymodel.URLHash,
) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(listedDocuments))
	for _, document := range listedDocuments {
		if !documents.contains(document) {
			continue
		}
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
}

func documentsWithoutRepeats(documents []yacymodel.URLHash) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(documents))
	seenDocuments := make(distinctDocuments, len(documents))
	for _, document := range documents {
		if seenDocuments.contains(document) {
			continue
		}
		seenDocuments.add(document)
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
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
