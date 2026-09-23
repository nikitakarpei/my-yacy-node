package wordjoined

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func urlMetadataAsksFor(
	documentsWithoutMetadataMostHeldFirst []yacymodel.URLHash,
	answeredSearchDocumentsAsks []peerasks.AnsweredSearchDocumentsAsk,
	partitions yacymodel.DHTRingPartitions,
	urlMetadataAskDocumentsCeiling int,
) []peerasks.URLMetadataAsk {
	peers := peersWithTheirAbstractsFrom(answeredSearchDocumentsAsks)
	var asks []peerasks.URLMetadataAsk
	for partition, documents := range documentsPerPartitionFrom(
		documentsWithoutMetadataMostHeldFirst, partitions,
	) {
		if len(documents) == 0 {
			continue
		}
		askDocuments := documents[:min(len(documents), urlMetadataAskDocumentsCeiling)]
		for _, peer := range peers.holdersOfTheMostFirstAmong(askDocuments) {
			asks = append(asks, peerasks.URLMetadataAsk{
				Peer:      peer,
				Partition: uint(partition),
				Documents: askDocuments,
			})
		}
	}

	return asks
}

type peersWithTheirAbstracts []peerWithItsAbstracts

type peerWithItsAbstracts struct {
	askablePeer             peerdirectory.AskablePeer
	documentsInItsAbstracts distinctDocuments
}

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

func (peers peersWithTheirAbstracts) holdersOfTheMostFirstAmong(
	documents []yacymodel.URLHash,
) []peerdirectory.AskablePeer {
	type holder struct {
		askablePeer  peerdirectory.AskablePeer
		amountOfHeld int
	}
	var holders []holder
	for _, peer := range peers {
		amountOfHeld := peer.amountOfDocumentsHeldAmong(documents)
		if amountOfHeld == 0 {
			continue
		}
		holders = append(holders, holder{askablePeer: peer.askablePeer, amountOfHeld: amountOfHeld})
	}
	slices.SortStableFunc(holders, func(first, second holder) int {
		return cmp.Compare(second.amountOfHeld, first.amountOfHeld)
	})
	askablePeers := make([]peerdirectory.AskablePeer, 0, len(holders))
	for _, holder := range holders {
		askablePeers = append(askablePeers, holder.askablePeer)
	}

	return askablePeers
}

func (peer peerWithItsAbstracts) amountOfDocumentsHeldAmong(documents []yacymodel.URLHash) int {
	amount := 0
	for _, document := range documents {
		if !peer.documentsInItsAbstracts.contains(document) {
			continue
		}
		amount++
	}

	return amount
}
