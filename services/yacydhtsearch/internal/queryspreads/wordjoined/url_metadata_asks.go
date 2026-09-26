package wordjoined

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	ctx context.Context,
	documentsWithoutMetadataMostHeldFirst []yacymodel.URLHash,
	answers []wordpartitionasks.ReplicaAnswer,
	askCeilings URLMetadataAskCeilings,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	peers := peersWithTheirAbstractsFrom(answers)
	asksOfEachPeer := peers.urlMetadataAsks(
		ctx,
		documentsWithoutMetadataMostHeldFirst,
		askCeilings,
	)

	return asksCoveringMostDocuments(asksOfEachPeer, amountOfPeersHoldingOneWord)
}

type peersWithTheirAbstracts []peerWithItsAbstracts

func peersWithTheirAbstractsFrom(
	answers []wordpartitionasks.ReplicaAnswer,
) peersWithTheirAbstracts {
	peers := make(peersWithTheirAbstracts, 0, len(answers))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answer := range answers {
		place, placed := placeOfPeer[answer.Replica.Hash]
		if !placed {
			place = len(peers)
			placeOfPeer[answer.Replica.Hash] = place
			peers = append(peers, peerWithItsAbstracts{
				askablePeer:             answer.Replica,
				documentsInItsAbstracts: distinctDocuments{},
			})
		}
		for _, listedDocument := range answer.ListedDocuments {
			peers[place].documentsInItsAbstracts.add(listedDocument.Hash)
		}
	}

	return peers
}

func (peers peersWithTheirAbstracts) urlMetadataAsks(
	ctx context.Context,
	documentsMostHeldFirst []yacymodel.URLHash,
	askCeilings URLMetadataAskCeilings,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(peers))
	for _, peer := range peers {
		ask := peer.urlMetadataAsk(
			documentsMostHeldFirst, askCeilings.CeilingOf(ctx, peer.askablePeer.Address),
		)
		if len(ask.Documents) == 0 {
			continue
		}
		asks = append(asks, ask)
	}

	return asks
}

type peerWithItsAbstracts struct {
	askablePeer             peerdirectory.AskablePeer
	documentsInItsAbstracts distinctDocuments
}

func (peer peerWithItsAbstracts) urlMetadataAsk(
	documentsMostHeldFirst []yacymodel.URLHash,
	askCeiling int,
) peerasks.URLMetadataAsk {
	heldDocumentsMostHeldFirst := peer.documentsHeldAmong(documentsMostHeldFirst)

	return peerasks.URLMetadataAsk{
		Peer:      peer.askablePeer,
		Documents: chosenDocumentsAmong(heldDocumentsMostHeldFirst, askCeiling),
	}
}

func (peer peerWithItsAbstracts) documentsHeldAmong(
	documentsMostHeldFirst []yacymodel.URLHash,
) []yacymodel.URLHash {
	heldDocuments := make([]yacymodel.URLHash, 0, len(peer.documentsInItsAbstracts))
	for _, document := range documentsMostHeldFirst {
		if !peer.documentsInItsAbstracts.contains(document) {
			continue
		}
		heldDocuments = append(heldDocuments, document)
	}

	return heldDocuments
}

func chosenDocumentsAmong(
	heldDocumentsMostHeldFirst []yacymodel.URLHash,
	askCeiling int,
) []yacymodel.URLHash {
	if len(heldDocumentsMostHeldFirst) <= askCeiling {
		return heldDocumentsMostHeldFirst
	}
	heldDocumentsLeastHeldFirst := slices.Clone(heldDocumentsMostHeldFirst)
	slices.Reverse(heldDocumentsLeastHeldFirst)

	return heldDocumentsLeastHeldFirst[:askCeiling]
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
