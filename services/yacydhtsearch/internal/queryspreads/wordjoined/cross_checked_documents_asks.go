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
	peerStandings peerjudgements.PeerStandings,
	crossCheckedDocumentsCeiling int,
) []peerasks.CrossCheckedDocumentsAsk {
	asks := make(
		[]peerasks.CrossCheckedDocumentsAsk,
		0,
		len(candidates.ofPartlyListedWordPartitions),
	)
	for _, candidatesOfWordPartition := range candidates.ofPartlyListedWordPartitions {
		peersNotYetAskedToCrossCheck := peersNotYetAskedToCrossCheckAmong(
			peersNotIgnoringTheCrossCheckAmong(
				candidatesOfWordPartition.wordPartition.replicasThatDidNotListAllTheyHold(),
				peerStandings,
			),
			asks,
		)
		amountOfDocumentsToDeal := min(
			len(candidatesOfWordPartition.documents),
			len(peersNotYetAskedToCrossCheck)*crossCheckedDocumentsCeiling,
		)
		asks = append(asks, crossCheckedDocumentsAsksDealtAcross(
			peersNotYetAskedToCrossCheck,
			candidatesOfWordPartition.wordPartition.word,
			candidatesOfWordPartition.documents[:amountOfDocumentsToDeal],
		)...)
	}

	return asks
}

func peersThatMayCrossCheckIn(
	candidates crossCheckCandidates,
) []peerjudgements.PeerAtVersion {
	var peers []peerjudgements.PeerAtVersion
	for _, candidatesOfWordPartition := range candidates.ofPartlyListedWordPartitions {
		for _, replica := range candidatesOfWordPartition.wordPartition.replicasThatDidNotListAllTheyHold() {
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
	replicas []queryWordOnReplica,
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

func crossCheckedDocumentsAsksDealtAcross(
	peers []peerdirectory.AskablePeer,
	queryWord yacymodel.Hash,
	documents []yacymodel.URLHash,
) []peerasks.CrossCheckedDocumentsAsk {
	documentsDealtToEachPeer := make([][]yacymodel.URLHash, len(peers))
	for turn, document := range documents {
		place := turn % len(peers)
		documentsDealtToEachPeer[place] = append(documentsDealtToEachPeer[place], document)
	}

	asks := make([]peerasks.CrossCheckedDocumentsAsk, 0, len(peers))
	for place, documentsDealtToOnePeer := range documentsDealtToEachPeer {
		if len(documentsDealtToOnePeer) == 0 {
			continue
		}
		asks = append(asks, peerasks.CrossCheckedDocumentsAsk{
			Peer:      peers[place],
			Word:      queryWord,
			Documents: documentsDealtToOnePeer,
		})
	}

	return asks
}
