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
	peersWithTheirDocuments := peersWithListedDocumentsFrom(answeredMatchedAndHeldDocumentsAsks)
	asksOfEachPeer := peersWithTheirDocuments.urlMetadataAsks(
		documentsWithoutMetadataMostListedFirst,
		urlMetadataAskDocumentsCeiling,
	)

	return asksCoveringMostDocuments(asksOfEachPeer, amountOfPeersHoldingOneWord)
}

type peersWithListedDocuments []peerWithListedDocuments

func peersWithListedDocumentsFrom(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
) peersWithListedDocuments {
	peersWithTheirDocuments := make(peersWithListedDocuments, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		place, placed := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !placed {
			place = len(peersWithTheirDocuments)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			peersWithTheirDocuments = append(peersWithTheirDocuments, peerWithListedDocuments{
				askablePeer:     answeredAsk.Ask.Peer,
				listedDocuments: distinctDocuments{},
			})
		}
		for _, document := range answeredAsk.DocumentsListedForTheWord {
			peersWithTheirDocuments[place].listedDocuments.add(document)
		}
	}

	return peersWithTheirDocuments
}

func (peersWithTheirDocuments peersWithListedDocuments) urlMetadataAsks(
	documentsMostListedFirst []yacymodel.URLHash,
	urlMetadataAskDocumentsCeiling int,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(peersWithTheirDocuments))
	for _, peerWithItsDocuments := range peersWithTheirDocuments {
		ask := peerWithItsDocuments.urlMetadataAsk(
			documentsMostListedFirst, urlMetadataAskDocumentsCeiling,
		)
		if len(ask.Documents) == 0 {
			continue
		}
		asks = append(asks, ask)
	}

	return asks
}

type peerWithListedDocuments struct {
	askablePeer     peerdirectory.AskablePeer
	listedDocuments distinctDocuments
}

func (peerWithItsDocuments peerWithListedDocuments) urlMetadataAsk(
	documentsMostListedFirst []yacymodel.URLHash,
	urlMetadataAskDocumentsCeiling int,
) peerasks.URLMetadataAsk {
	askDocuments := make([]yacymodel.URLHash, 0, min(
		len(peerWithItsDocuments.listedDocuments), urlMetadataAskDocumentsCeiling,
	))
	for _, document := range documentsMostListedFirst {
		if len(askDocuments) == urlMetadataAskDocumentsCeiling {
			break
		}
		if !peerWithItsDocuments.listedDocuments.contains(document) {
			continue
		}
		askDocuments = append(askDocuments, document)
	}

	return peerasks.URLMetadataAsk{
		Peer:      peerWithItsDocuments.askablePeer,
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
