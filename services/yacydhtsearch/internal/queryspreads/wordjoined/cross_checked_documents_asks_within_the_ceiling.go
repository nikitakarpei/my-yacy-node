package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type crossCheckedDocumentsAsksWithinTheCeiling struct {
	asks                                         []peerasks.CrossCheckedDocumentsAsk
	documentsPastTheCrossCheckedDocumentsCeiling distinctDocuments
}

func crossCheckedDocumentsAsksWithinTheCeilingFor(
	queryWordsBesideTheLeadingQueryWord []queryWordAcrossReplicas,
	documentsListedByThePeersOfTheLeadingQueryWordMostListedFirst []yacymodel.URLHash,
	crossCheckedDocumentsCeiling int,
) crossCheckedDocumentsAsksWithinTheCeiling {
	asksWithinTheCeiling := crossCheckedDocumentsAsksWithinTheCeiling{
		documentsPastTheCrossCheckedDocumentsCeiling: distinctDocuments{},
	}
	for _, queryWord := range queryWordsBesideTheLeadingQueryWord {
		if queryWord.isFullyListed() {
			continue
		}
		candidateDocuments := documentsNotListedByPeersAmong(
			documentsListedByThePeersOfTheLeadingQueryWordMostListedFirst,
			queryWord.documentsListedByPeers(),
		)
		peersNotYetAskedToCrossCheck := peersNotYetAskedToCrossCheckAmong(
			queryWord.peersThatDidNotListAllTheyHold(),
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
		for _, document := range candidateDocuments[amountOfDocumentsToDeal:] {
			asksWithinTheCeiling.documentsPastTheCrossCheckedDocumentsCeiling.add(document)
		}
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
