package urlmetadataasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func (asker Asker) asksOfEachPeerAmong(
	ctx context.Context,
	peers []documentholders.PeerWithItsDocuments,
	documentsMostHeldFirst []yacymodel.URLHash,
) []peerasks.URLMetadataAsk {
	asks := make([]peerasks.URLMetadataAsk, 0, len(peers))
	for _, peer := range peers {
		ask := urlMetadataAskOf(
			peer, documentsMostHeldFirst, asker.ceilings.CeilingOf(ctx, peer.Peer.Address),
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
