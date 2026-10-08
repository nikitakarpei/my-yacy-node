package urlmetadataasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askLimits struct {
	ceilings        Ceilings
	asksPerDocument int
}

func (limits askLimits) asksFor(
	ctx context.Context,
	holders documentholders.Holders,
) []peerasks.URLMetadataAsk {
	asksOfEachPeer := limits.asksOfEachPeerAmong(
		ctx, holders.PeersWithTheirDocuments(), holders.MostHeldFirst(),
	)

	return limits.coveringAsksAmong(asksOfEachPeer)
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

func (limits askLimits) coveringAsksAmong(
	asks []peerasks.URLMetadataAsk,
) []peerasks.URLMetadataAsk {
	chosenPlaces := map[int]struct{}{}
	asksNamingEachDocument := map[yacymodel.URLHash]int{}
	for {
		place, found := limits.placeOfMostCoveringAskAmong(
			asks,
			chosenPlaces,
			asksNamingEachDocument,
		)
		if !found {
			return asksAt(asks, chosenPlaces)
		}
		chosenPlaces[place] = struct{}{}
		for _, document := range asks[place].Documents {
			asksNamingEachDocument[document]++
		}
	}
}

func (limits askLimits) placeOfMostCoveringAskAmong(
	asks []peerasks.URLMetadataAsk,
	chosenPlaces map[int]struct{},
	asksNamingEachDocument map[yacymodel.URLHash]int,
) (int, bool) {
	placeOfMostCoveringAsk := 0
	mostUncoveredDocuments := 0
	for place, ask := range asks {
		if _, chosen := chosenPlaces[place]; chosen {
			continue
		}
		amountOfUncoveredDocuments := limits.amountOfDocumentsNotCovered(
			ask.Documents, asksNamingEachDocument,
		)
		if amountOfUncoveredDocuments > mostUncoveredDocuments {
			placeOfMostCoveringAsk = place
			mostUncoveredDocuments = amountOfUncoveredDocuments
		}
	}

	return placeOfMostCoveringAsk, mostUncoveredDocuments > 0
}

func (limits askLimits) amountOfDocumentsNotCovered(
	documents []yacymodel.URLHash,
	asksNamingEachDocument map[yacymodel.URLHash]int,
) int {
	amount := 0
	for _, document := range documents {
		if asksNamingEachDocument[document] >= limits.asksPerDocument {
			continue
		}
		amount++
	}

	return amount
}

func asksAt(asks []peerasks.URLMetadataAsk, places map[int]struct{}) []peerasks.URLMetadataAsk {
	asksAtThePlaces := make([]peerasks.URLMetadataAsk, 0, len(places))
	for place, ask := range asks {
		if _, kept := places[place]; !kept {
			continue
		}
		asksAtThePlaces = append(asksAtThePlaces, ask)
	}

	return asksAtThePlaces
}
