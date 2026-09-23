package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func crossCheckAsksFor(
	candidates crossCheckCandidates,
	peersAskedToDiscover map[yacymodel.Hash]struct{},
	peerStandings peerjudgements.PeerStandings,
	documentsToMatchCeiling int,
	peerItemsCeiling int,
) []peerasks.SearchDocumentsAsk {
	var asks []peerasks.SearchDocumentsAsk
	for _, candidatesOfWordPartition := range candidates.ofWordPartitionsWithoutACompleteAbstract {
		peersThatMayCrossCheck := peersNotYetAskedToCrossCheckAmong(
			peersUsefulForTheCrossCheckAmong(
				peersNotAskedToDiscoverAmong(
					candidatesOfWordPartition.wordPartition.replicas,
					peersAskedToDiscover,
				),
				peerStandings,
			),
			asks,
		)
		for _, peer := range peersMostUsefulForTheCrossCheckFirst(peersThatMayCrossCheck, peerStandings) {
			asks = append(asks, peerasks.SearchDocumentsAsk{
				Peer:      peer,
				Partition: candidatesOfWordPartition.wordPartition.partition,
				Word:      candidatesOfWordPartition.wordPartition.word,
				DocumentsToMatch: candidatesOfWordPartition.mostHeldDocumentsUpTo(
					documentsToMatchCeiling,
				),
				ItemsCeiling: peerItemsCeiling,
			})
		}
	}

	return asks
}

func peersNotAskedToDiscoverAmong(
	replicas []wordReplica,
	peersAskedToDiscover map[yacymodel.Hash]struct{},
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(replicas))
	for _, replica := range replicas {
		if _, asked := peersAskedToDiscover[replica.peer.Hash]; asked {
			continue
		}
		keptPeers = append(keptPeers, replica.peer)
	}

	return keptPeers
}

func peersNotYetAskedToCrossCheckAmong(
	peers []peerdirectory.AskablePeer,
	asks []peerasks.SearchDocumentsAsk,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(peers))
	for _, peer := range peers {
		if slices.ContainsFunc(asks, func(ask peerasks.SearchDocumentsAsk) bool {
			return ask.Peer.Hash == peer.Hash
		}) {
			continue
		}
		keptPeers = append(keptPeers, peer)
	}

	return keptPeers
}
