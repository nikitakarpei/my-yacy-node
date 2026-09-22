package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type crossCheckedDocumentsAsksWithinTheCeiling struct {
	asks                                                 []peerasks.CrossCheckedDocumentsAsk
	amountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
}

func crossCheckedDocumentsAsksWithinTheCeilingFor(
	queryWordsBesideTheLeadingQueryWord []queryWordAcrossReplicas,
	documentsListedByThePeersOfTheLeadingQueryWordMostListedFirst []yacymodel.URLHash,
	standings []peerjudgements.PeerStanding,
	crossCheckedDocumentsCeiling int,
) crossCheckedDocumentsAsksWithinTheCeiling {
	asksWithinTheCeiling := crossCheckedDocumentsAsksWithinTheCeiling{}
	for _, queryWord := range queryWordsBesideTheLeadingQueryWord {
		if queryWord.isFullyListed() {
			continue
		}
		candidateDocuments := documentsNotListedByPeersAmong(
			documentsListedByThePeersOfTheLeadingQueryWordMostListedFirst,
			queryWord.documentsListedByPeers(),
		)
		peersNotYetAskedToCrossCheck := peersNotYetAskedToCrossCheckAmong(
			peersNotIgnoringAmong(queryWord.replicasThatDidNotListAllTheyHold(), standings),
			asksWithinTheCeiling.asks,
		)
		amountOfDocumentsToDeal := min(
			len(candidateDocuments), len(peersNotYetAskedToCrossCheck)*crossCheckedDocumentsCeiling,
		)
		asksWithinTheCeiling.asks = append(
			asksWithinTheCeiling.asks,
			crossCheckedDocumentsAsksDealtAcross(
				peersNotYetAskedToCrossCheck,
				queryWord.word,
				candidateDocuments[:amountOfDocumentsToDeal],
			)...)
		asksWithinTheCeiling.amountOfDocumentsPastTheCrossCheckedDocumentsCeiling += len(
			candidateDocuments,
		) - amountOfDocumentsToDeal
	}

	return asksWithinTheCeiling
}

func documentsNotListedByPeersAmong(
	documents []yacymodel.URLHash,
	documentsListedByPeers distinctDocuments,
) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(documents))
	for _, document := range documents {
		if documentsListedByPeers.contains(document) {
			continue
		}
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
}

func peersNotIgnoringAmong(
	replicas []queryWordOnReplica,
	standings []peerjudgements.PeerStanding,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(replicas))
	for _, replica := range replicas {
		if standingOf(replica.peer, standings) == peerjudgements.Ignoring {
			continue
		}
		keptPeers = append(keptPeers, replica.peer)
	}

	return keptPeers
}

func standingOf(
	peer peerdirectory.AskablePeer,
	standings []peerjudgements.PeerStanding,
) peerjudgements.Standing {
	place := slices.IndexFunc(standings, func(standing peerjudgements.PeerStanding) bool {
		return standing.Peer == peer.Hash
	})
	if place < 0 {
		return peerjudgements.NeverJudged
	}

	return standings[place].Standing
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
