package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	documentsWithoutMetadataMostListedFirst []yacymodel.URLHash,
	answeredSearchDocumentsAsks []peerasks.AnsweredSearchDocumentsAsk,
	urlMetadataAskDocumentsCeiling int,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	peers := peersWithListedDocumentsFrom(answeredSearchDocumentsAsks)
	asksOfEachPeer := peers.urlMetadataAsks(
		documentsWithoutMetadataMostListedFirst,
		urlMetadataAskDocumentsCeiling,
	)

	return asksCoveringMostDocuments(asksOfEachPeer, amountOfPeersHoldingOneWord)
}

type peersWithListedDocuments []peerWithListedDocuments

func peersWithListedDocumentsFrom(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) peersWithListedDocuments {
	peers := make(peersWithListedDocuments, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		place, placed := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !placed {
			place = len(peers)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			peers = append(peers, peerWithListedDocuments{
				askablePeer:     answeredAsk.Ask.Peer,
				listedDocuments: distinctDocuments{},
			})
		}
		for _, document := range answeredAsk.DocumentsListedForTheWord {
			peers[place].listedDocuments.add(document)
		}
	}

	return peers
}

func (peers peersWithListedDocuments) urlMetadataAsks(
	documentsMostListedFirst []yacymodel.URLHash,
	urlMetadataAskDocumentsCeiling int,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(peers))
	for _, peer := range peers {
		ask := peer.urlMetadataAsk(
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

func (peer peerWithListedDocuments) urlMetadataAsk(
	documentsMostListedFirst []yacymodel.URLHash,
	urlMetadataAskDocumentsCeiling int,
) peerasks.URLMetadataAsk {
	askDocuments := make([]yacymodel.URLHash, 0, min(
		len(peer.listedDocuments), urlMetadataAskDocumentsCeiling,
	))
	for _, document := range documentsMostListedFirst {
		if len(askDocuments) == urlMetadataAskDocumentsCeiling {
			break
		}
		if !peer.listedDocuments.contains(document) {
			continue
		}
		askDocuments = append(askDocuments, document)
	}

	return peerasks.URLMetadataAsk{
		Peer:      peer.askablePeer,
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
	for len(coveringAsks) < amountOfPeersHoldingOneWord {
		place, found := placeOfMostCoveringAskAmong(asks, coveredDocuments)
		if !found {
			break
		}
		coveringAsks = append(coveringAsks, asks[place])
		for _, document := range asks[place].Documents {
			coveredDocuments.add(document)
		}
	}

	return coveringAsks
}

func placeOfMostCoveringAskAmong(
	asks []peerasks.URLMetadataAsk,
	coveredDocuments distinctDocuments,
) (int, bool) {
	placeOfMostCoveringAsk := 0
	mostUncoveredDocuments := 0
	for place, ask := range asks {
		amountOfUncoveredDocuments := amountOfDocumentsNotCovered(ask.Documents, coveredDocuments)
		if amountOfUncoveredDocuments > mostUncoveredDocuments {
			placeOfMostCoveringAsk = place
			mostUncoveredDocuments = amountOfUncoveredDocuments
		}
	}

	return placeOfMostCoveringAsk, mostUncoveredDocuments > 0
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
