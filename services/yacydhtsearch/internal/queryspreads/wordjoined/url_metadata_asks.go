package wordjoined

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	ctx context.Context,
	documentsWithoutMetadataMostHeldFirst []yacymodel.URLHash,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
	askCeilings URLMetadataAskCeilings,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	peers := peersWithTheirAbstractsFrom(answeredAsks)
	asksOfEachPeer := peers.urlMetadataAsks(
		ctx,
		documentsWithoutMetadataMostHeldFirst,
		askCeilings,
	)

	return asksCoveringMostDocuments(asksOfEachPeer, amountOfPeersHoldingOneWord)
}

type peersWithTheirAbstracts []peerWithItsAbstracts

func peersWithTheirAbstractsFrom(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) peersWithTheirAbstracts {
	peers := make(peersWithTheirAbstracts, 0, len(answeredAsks))
	placeOfPeer := map[yacymodel.Hash]int{}
	for _, answeredAsk := range answeredAsks {
		place, placed := placeOfPeer[answeredAsk.Ask.Peer.Hash]
		if !placed {
			place = len(peers)
			placeOfPeer[answeredAsk.Ask.Peer.Hash] = place
			peers = append(peers, peerWithItsAbstracts{
				askablePeer:             answeredAsk.Ask.Peer,
				documentsInItsAbstracts: distinctDocuments{},
			})
		}
		for _, document := range answeredAsk.Abstract {
			peers[place].documentsInItsAbstracts.add(document)
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
	askDocuments := make([]yacymodel.URLHash, 0, min(
		len(peer.documentsInItsAbstracts), askCeiling,
	))
	for _, document := range documentsMostHeldFirst {
		if len(askDocuments) == askCeiling {
			break
		}
		if !peer.documentsInItsAbstracts.contains(document) {
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
