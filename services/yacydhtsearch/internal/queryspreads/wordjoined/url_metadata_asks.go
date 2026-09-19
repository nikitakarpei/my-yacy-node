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
	documentsListedByEachPeer := documentsListedPerPeerFrom(answeredMatchedAndHeldDocumentsAsks)
	asks := documentsListedByEachPeer.urlMetadataAsks(
		documentsWithoutMetadataMostListedFirst,
		urlMetadataAskDocumentsCeiling,
	)

	return asksCoveringMostDocuments(asks, amountOfPeersHoldingOneWord)
}

type documentsListedPerPeer []documentsListedByPeer

func documentsListedPerPeerFrom(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) documentsListedPerPeer {
	documentsListedByEachPeer := make(documentsListedPerPeer, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		place, placed := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !placed {
			place = len(documentsListedByEachPeer)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			documentsListedByEachPeer = append(documentsListedByEachPeer, documentsListedByPeer{
				peer:      answeredAsk.Ask.Peer,
				documents: distinctDocuments{},
			})
		}
		for _, document := range answeredAsk.DocumentsListedForTheWord {
			documentsListedByEachPeer[place].documents.add(document)
		}
	}

	return documentsListedByEachPeer
}

func (documentsListedByEachPeer documentsListedPerPeer) urlMetadataAsks(
	documentsMostListedFirst []yacymodel.URLHash,
	urlMetadataAskDocumentsCeiling int,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(documentsListedByEachPeer))
	for _, documentsListedByOnePeer := range documentsListedByEachPeer {
		ask := documentsListedByOnePeer.urlMetadataAsk(
			documentsMostListedFirst, urlMetadataAskDocumentsCeiling,
		)
		if len(ask.Documents) == 0 {
			continue
		}
		asks = append(asks, ask)
	}

	return asks
}

type documentsListedByPeer struct {
	peer      peerdirectory.AskablePeer
	documents distinctDocuments
}

func (documentsListedByOnePeer documentsListedByPeer) urlMetadataAsk(
	documentsMostListedFirst []yacymodel.URLHash,
	urlMetadataAskDocumentsCeiling int,
) peerasks.URLMetadataAsk {
	askDocuments := make([]yacymodel.URLHash, 0, min(
		len(documentsListedByOnePeer.documents), urlMetadataAskDocumentsCeiling,
	))
	for _, document := range documentsMostListedFirst {
		if len(askDocuments) == urlMetadataAskDocumentsCeiling {
			break
		}
		if !documentsListedByOnePeer.documents.contains(document) {
			continue
		}
		askDocuments = append(askDocuments, document)
	}

	return peerasks.URLMetadataAsk{
		Peer:      documentsListedByOnePeer.peer,
		Documents: askDocuments,
	}
}

func asksCoveringMostDocuments(
	asks []peerasks.URLMetadataAsk,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	if len(asks) <= amountOfPeersHoldingOneWord {
		return asks
	}

	coveringAsks := make([]peerasks.URLMetadataAsk, 0, amountOfPeersHoldingOneWord)
	coveredDocuments := distinctDocuments{}
	takenAsks := make([]bool, len(asks))
	for len(coveringAsks) < amountOfPeersHoldingOneWord {
		mostCoveringAsk := mostCoveringAskAmong(asks, takenAsks, coveredDocuments)
		if mostCoveringAsk.amountOfUncoveredDocuments == 0 {
			break
		}
		takenAsks[mostCoveringAsk.place] = true
		for _, document := range asks[mostCoveringAsk.place].Documents {
			coveredDocuments.add(document)
		}
		coveringAsks = append(coveringAsks, asks[mostCoveringAsk.place])
	}

	return coveringAsks
}

type mostCoveringAsk struct {
	place                      int
	amountOfUncoveredDocuments int
}

func mostCoveringAskAmong(
	asks []peerasks.URLMetadataAsk,
	takenAsks []bool,
	coveredDocuments distinctDocuments,
) mostCoveringAsk {
	mostCoveringAsk := mostCoveringAsk{}
	for place, ask := range asks {
		if takenAsks[place] {
			continue
		}
		amountOfUncoveredDocuments := amountOfDocumentsNotCovered(ask.Documents, coveredDocuments)
		if amountOfUncoveredDocuments > mostCoveringAsk.amountOfUncoveredDocuments {
			mostCoveringAsk.place = place
			mostCoveringAsk.amountOfUncoveredDocuments = amountOfUncoveredDocuments
		}
	}

	return mostCoveringAsk
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
