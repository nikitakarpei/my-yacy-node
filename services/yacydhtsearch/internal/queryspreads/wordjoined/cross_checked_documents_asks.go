package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func crossCheckedDocumentsAsksFor(
	partlyListedQueryWords []queryWordAcrossReplicas,
	documentsOfTheLeadingQueryWordMostListedFirst []yacymodel.URLHash,
	peerStandings []peerjudgements.PeerStanding,
	crossCheckedDocumentsCeiling int,
) []peerasks.CrossCheckedDocumentsAsk {
	asks := make([]peerasks.CrossCheckedDocumentsAsk, 0, len(partlyListedQueryWords))
	for _, queryWord := range partlyListedQueryWords {
		candidateDocuments := queryWord.crossCheckCandidatesAmong(
			documentsOfTheLeadingQueryWordMostListedFirst,
		)
		peersNotYetAskedToCrossCheck := peersNotYetAskedToCrossCheckAmong(
			peersNotIgnoringAmong(queryWord.replicasThatDidNotListAllTheyHold(), peerStandings),
			asks,
		)
		amountOfDocumentsToDeal := min(
			len(candidateDocuments), len(peersNotYetAskedToCrossCheck)*crossCheckedDocumentsCeiling,
		)
		asks = append(asks, crossCheckedDocumentsAsksDealtAcross(
			peersNotYetAskedToCrossCheck,
			queryWord.word,
			candidateDocuments[:amountOfDocumentsToDeal],
		)...)
	}

	return asks
}

func peersNotIgnoringAmong(
	replicas []queryWordOnReplica,
	peerStandings []peerjudgements.PeerStanding,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(replicas))
	for _, replica := range replicas {
		if standingOf(replica.peer, peerStandings) == peerjudgements.Ignoring {
			continue
		}
		keptPeers = append(keptPeers, replica.peer)
	}

	return keptPeers
}

func standingOf(
	peer peerdirectory.AskablePeer,
	peerStandings []peerjudgements.PeerStanding,
) peerjudgements.Standing {
	place := slices.IndexFunc(peerStandings, func(peerStanding peerjudgements.PeerStanding) bool {
		return peerStanding.Peer == peer.Hash
	})

	return peerStandings[place].Standing
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
