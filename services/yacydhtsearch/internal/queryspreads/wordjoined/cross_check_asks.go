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
	peerStandings peerjudgements.PeerStandings,
	documentsToMatchCeiling int,
) []peerasks.CrossCheckedDocumentsAsk {
	asks := make(
		[]peerasks.CrossCheckedDocumentsAsk,
		0,
		len(candidates.ofWordPartitionsWithoutACompleteAbstract),
	)
	for _, candidatesOfWordPartition := range candidates.ofWordPartitionsWithoutACompleteAbstract {
		peersNotYetAskedToCrossCheck := peersNotYetAskedToCrossCheckAmong(
			peersNotIgnoringTheCrossCheckAmong(
				candidatesOfWordPartition.wordPartition.replicasWithoutACompleteAbstract(),
				peerStandings,
			),
			asks,
		)
		asks = append(asks, crossCheckAsksOfEach(
			peersNotYetAskedToCrossCheck,
			candidatesOfWordPartition.wordPartition.word,
			candidatesOfWordPartition.mostHeldDocumentsUpTo(documentsToMatchCeiling),
		)...)
	}

	return asks
}

func peersThatMayCrossCheckIn(
	candidates crossCheckCandidates,
) []peerjudgements.PeerAtVersion {
	var peers []peerjudgements.PeerAtVersion
	for _, candidatesOfWordPartition := range candidates.ofWordPartitionsWithoutACompleteAbstract {
		for _, replica := range candidatesOfWordPartition.wordPartition.replicasWithoutACompleteAbstract() {
			if slices.ContainsFunc(peers, func(peer peerjudgements.PeerAtVersion) bool {
				return peer.Peer == replica.peer.Hash
			}) {
				continue
			}
			peers = append(peers, peerjudgements.PeerAtVersion{
				Peer:    replica.peer.Hash,
				Version: replica.versionClaimed(),
			})
		}
	}

	return peers
}

func peersNotIgnoringTheCrossCheckAmong(
	replicas []wordReplica,
	peerStandings peerjudgements.PeerStandings,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(replicas))
	for _, replica := range replicas {
		if peerStandings.StandingOf(replica.peer.Hash) == peerjudgements.Ignoring {
			continue
		}
		keptPeers = append(keptPeers, replica.peer)
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

func crossCheckAsksOfEach(
	peers []peerdirectory.AskablePeer,
	queryWord yacymodel.Hash,
	documents []yacymodel.URLHash,
) []peerasks.CrossCheckedDocumentsAsk {
	asks := make([]peerasks.CrossCheckedDocumentsAsk, 0, len(peers))
	for _, peer := range peers {
		asks = append(asks, peerasks.CrossCheckedDocumentsAsk{
			Peer:      peer,
			Word:      queryWord,
			Documents: documents,
		})
	}

	return asks
}
