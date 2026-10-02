package urlmetadataasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

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
				documentsInItsAbstracts: yacymodel.URLHashes{},
			})
		}
		for _, listedDocument := range answer.ListedDocuments {
			peers[place].documentsInItsAbstracts.Add(listedDocument.Hash)
		}
	}

	return peers
}

func (peers peersWithTheirAbstracts) asksFor(
	ctx context.Context,
	documentsMostHeldFirst []yacymodel.URLHash,
	ceilings Ceilings,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(peers))
	for _, peer := range peers {
		ask := peer.urlMetadataAsk(
			documentsMostHeldFirst, ceilings.CeilingOf(ctx, peer.askablePeer.Address),
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
	documentsInItsAbstracts yacymodel.URLHashes
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
		if !peer.documentsInItsAbstracts.Contains(document) {
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
