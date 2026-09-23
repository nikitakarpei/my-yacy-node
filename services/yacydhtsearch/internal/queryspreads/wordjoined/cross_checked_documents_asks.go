package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func crossCheckedDocumentsAsksFor(
	candidates crossCheckCandidates,
	peersAskedInTheFirstRound map[yacymodel.Hash]struct{},
	peerStandings peerjudgements.PeerStandings,
	crossCheckedDocumentsCeiling int,
	peerItemsCeiling int,
) []peerasks.CrossCheckedDocumentsAsk {
	var asks []peerasks.CrossCheckedDocumentsAsk
	for _, candidatesOfWordPartition := range candidates.ofPartlyListedWordPartitions {
		peersThatMayCrossCheck := peersNotYetAskedToCrossCheckAmong(
			peersNotIgnoringTheCrossCheckAmong(
				peersNotAskedInTheFirstRoundAmong(
					candidatesOfWordPartition.wordPartition.replicas, peersAskedInTheFirstRound,
				),
				peerStandings,
			),
			asks,
		)
		for _, peer := range peersInSecondRoundOrder(peersThatMayCrossCheck, peerStandings) {
			asks = append(asks, peerasks.CrossCheckedDocumentsAsk{
				Peer:      peer,
				Partition: candidatesOfWordPartition.wordPartition.partition,
				Word:      candidatesOfWordPartition.wordPartition.word,
				Documents: candidatesOfWordPartition.mostListedDocumentsUpTo(
					crossCheckedDocumentsCeiling,
				),
				ItemsCeiling: peerItemsCeiling,
			})
		}
	}

	return asks
}

func peersNotAskedInTheFirstRoundAmong(
	replicas []wordReplica,
	peersAskedInTheFirstRound map[yacymodel.Hash]struct{},
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(replicas))
	for _, replica := range replicas {
		if _, asked := peersAskedInTheFirstRound[replica.peer.Hash]; asked {
			continue
		}
		keptPeers = append(keptPeers, replica.peer)
	}

	return keptPeers
}

func peersNotIgnoringTheCrossCheckAmong(
	peers []peerdirectory.AskablePeer,
	peerStandings peerjudgements.PeerStandings,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(peers))
	for _, peer := range peers {
		if peerStandings.StandingOf(peer.Hash) == peerjudgements.Ignoring {
			continue
		}
		keptPeers = append(keptPeers, peer)
	}

	return keptPeers
}

func peersNotYetAskedToCrossCheckAmong(
	peers []peerdirectory.AskablePeer,
	asks []peerasks.CrossCheckedDocumentsAsk,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(peers))
	for _, peer := range peers {
		if slices.ContainsFunc(asks, func(ask peerasks.CrossCheckedDocumentsAsk) bool {
			return ask.Peer.Hash == peer.Hash
		}) {
			continue
		}
		keptPeers = append(keptPeers, peer)
	}

	return keptPeers
}
