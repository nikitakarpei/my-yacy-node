package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type heldDocumentsDeal struct {
	asks                    []peerasks.HeldDocumentsAsk
	documentsPastTheCeiling map[yacymodel.URLHash]struct{}
}

func heldDocumentsDealFor(
	cutOffQueryWords []answeredQueryWord,
	anchorDocumentsMostHeldFirst []yacymodel.URLHash,
	heldDocumentsCeiling int,
) heldDocumentsDeal {
	deal := heldDocumentsDeal{documentsPastTheCeiling: map[yacymodel.URLHash]struct{}{}}
	for _, cutOffQueryWord := range cutOffQueryWords {
		candidateDocuments := documentsNotHeldAmong(
			anchorDocumentsMostHeldFirst, cutOffQueryWord.documentsHeld(),
		)
		peersWithoutAnAsk := peersWithoutAnAskAmong(
			cutOffQueryWord.cutOffOrSilentPeers(),
			deal.asks,
		)
		amountOfDocumentsDealt := min(
			len(candidateDocuments), len(peersWithoutAnAsk)*heldDocumentsCeiling,
		)
		deal.asks = append(deal.asks, heldDocumentsAsksDealtAcross(
			peersWithoutAnAsk, cutOffQueryWord.word, candidateDocuments[:amountOfDocumentsDealt],
		)...)
		for _, document := range candidateDocuments[amountOfDocumentsDealt:] {
			deal.documentsPastTheCeiling[document] = struct{}{}
		}
	}

	return deal
}

func documentsNotHeldAmong(
	documents []yacymodel.URLHash,
	documentsHeld map[yacymodel.URLHash]struct{},
) []yacymodel.URLHash {
	keptDocuments := make([]yacymodel.URLHash, 0, len(documents))
	for _, document := range documents {
		if _, held := documentsHeld[document]; held {
			continue
		}
		keptDocuments = append(keptDocuments, document)
	}

	return keptDocuments
}

func peersWithoutAnAskAmong(
	peers []peerdirectory.AskablePeer,
	asks []peerasks.HeldDocumentsAsk,
) []peerdirectory.AskablePeer {
	keptPeers := make([]peerdirectory.AskablePeer, 0, len(peers))
	for _, peer := range peers {
		if slices.ContainsFunc(asks, func(ask peerasks.HeldDocumentsAsk) bool {
			return ask.Peer.Hash == peer.Hash
		}) {
			continue
		}
		keptPeers = append(keptPeers, peer)
	}

	return keptPeers
}

func heldDocumentsAsksDealtAcross(
	peers []peerdirectory.AskablePeer,
	queryWord yacymodel.Hash,
	documents []yacymodel.URLHash,
) []peerasks.HeldDocumentsAsk {
	documentsDealtToEachPeer := make([][]yacymodel.URLHash, len(peers))
	for turn, document := range documents {
		place := turn % len(peers)
		documentsDealtToEachPeer[place] = append(documentsDealtToEachPeer[place], document)
	}

	asks := make([]peerasks.HeldDocumentsAsk, 0, len(peers))
	for place, documentsDealtToOnePeer := range documentsDealtToEachPeer {
		if len(documentsDealtToOnePeer) == 0 {
			continue
		}
		asks = append(asks, peerasks.HeldDocumentsAsk{
			Peer:      peers[place],
			Word:      queryWord,
			Documents: documentsDealtToOnePeer,
		})
	}

	return asks
}
