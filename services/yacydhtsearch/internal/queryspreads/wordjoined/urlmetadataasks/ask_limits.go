package urlmetadataasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askLimits struct {
	ceilings          Ceilings
	networkRedundancy int
}

func (limits askLimits) asksFor(
	ctx context.Context,
	holders documentholders.Holders,
) []peerasks.URLMetadataAsk {
	asksOfEachPeer := limits.asksOfEachPeerAmong(
		ctx, holders.PeersWithTheirDocuments(), holders.MostHeldFirst(),
	)

	return asksCoveringMostDocuments(asksOfEachPeer, limits.networkRedundancy)
}

func (limits askLimits) asksOfEachPeerAmong(
	ctx context.Context,
	peers []documentholders.PeerWithItsDocuments,
	documentsMostHeldFirst []yacymodel.URLHash,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(peers))
	for _, peer := range peers {
		ask := urlMetadataAskOf(
			peer, documentsMostHeldFirst, limits.ceilings.CeilingOf(ctx, peer.Peer.Address),
		)
		if len(ask.Documents) == 0 {
			continue
		}
		asks = append(asks, ask)
	}

	return asks
}

func urlMetadataAskOf(
	peer documentholders.PeerWithItsDocuments,
	documentsMostHeldFirst []yacymodel.URLHash,
	askCeiling int,
) peerasks.URLMetadataAsk {
	heldDocumentsMostHeldFirst := documentsHeldAmong(documentsMostHeldFirst, peer.Documents)

	return peerasks.URLMetadataAsk{
		Peer:      peer.Peer,
		Documents: chosenDocumentsAmong(heldDocumentsMostHeldFirst, askCeiling),
	}
}

func documentsHeldAmong(
	documentsMostHeldFirst []yacymodel.URLHash,
	documentsOfThePeer yacymodel.URLHashes,
) []yacymodel.URLHash {
	heldDocuments := make([]yacymodel.URLHash, 0, len(documentsOfThePeer))
	for _, document := range documentsMostHeldFirst {
		if !documentsOfThePeer.Contains(document) {
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
	networkRedundancy int,
) []peerasks.URLMetadataAsk {
	if len(asks) <= networkRedundancy {
		return asks
	}

	coveringAsks := make([]peerasks.URLMetadataAsk, 0, networkRedundancy)
	coveredDocuments := yacymodel.URLHashes{}
	for len(coveringAsks) < networkRedundancy {
		place, found := placeOfMostCoveringAskAmong(asks, coveredDocuments)
		if !found {
			break
		}
		coveringAsks = append(coveringAsks, asks[place])
		coveredDocuments.AddEach(asks[place].Documents)
	}

	return coveringAsks
}

func placeOfMostCoveringAskAmong(
	asks []peerasks.URLMetadataAsk,
	coveredDocuments yacymodel.URLHashes,
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
	coveredDocuments yacymodel.URLHashes,
) int {
	amount := 0
	for _, document := range documents {
		if coveredDocuments.Contains(document) {
			continue
		}
		amount++
	}

	return amount
}
