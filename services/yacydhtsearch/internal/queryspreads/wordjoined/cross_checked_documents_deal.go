package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type crossCheckedDocumentsDeal struct {
	asks                    []peerasks.CrossCheckedDocumentsAsk
	documentsPastTheCeiling map[yacymodel.URLHash]struct{}
}

func crossCheckedDocumentsDealFor(
	queryWordsBesideTheLeadingQueryWord []answeredQueryWord,
	documentsListedByThePeersOfTheLeadingQueryWordMostListedFirst []yacymodel.URLHash,
	crossCheckedDocumentsCeiling int,
) crossCheckedDocumentsDeal {
	deal := crossCheckedDocumentsDeal{documentsPastTheCeiling: map[yacymodel.URLHash]struct{}{}}
	for _, queryWord := range queryWordsBesideTheLeadingQueryWord {
		if queryWord.isFullyListed() {
			continue
		}
		candidateDocuments := documentsNotListedAmong(
			documentsListedByThePeersOfTheLeadingQueryWordMostListedFirst,
			queryWord.documentsListed(),
		)
		peersWithoutAnAsk := peersWithoutAnAskAmong(
			queryWord.peersThatDidNotListAllTheyHold(),
			deal.asks,
		)
		amountOfDocumentsDealt := min(
			len(candidateDocuments), len(peersWithoutAnAsk)*crossCheckedDocumentsCeiling,
		)
		deal.asks = append(deal.asks, crossCheckedDocumentsAsksDealtAcross(
			peersWithoutAnAsk, queryWord.word, candidateDocuments[:amountOfDocumentsDealt],
		)...)
		for _, document := range candidateDocuments[amountOfDocumentsDealt:] {
			deal.documentsPastTheCeiling[document] = struct{}{}
		}
	}

	return deal
}

func documentsNotListedAmong(
	documents []yacymodel.URLHash,
	documentsListed map[yacymodel.URLHash]struct{},
) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(documents))
	for _, document := range documents {
		if _, listed := documentsListed[document]; listed {
			continue
		}
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
}

func peersWithoutAnAskAmong(
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
