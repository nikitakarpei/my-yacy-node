package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func judgeAskedPeersIn(round crossCheckedDocumentsRound) []peerjudgements.JudgedPeer {
	judgedPeers := make([]peerjudgements.JudgedPeer, 0, len(round.asks))
	for _, ask := range round.asks {
		judgedPeers = append(judgedPeers, judgeAskedPeer(ask, round.answeredAsks))
	}

	return judgedPeers
}

func judgeAskedPeer(
	ask peerasks.CrossCheckedDocumentsAsk,
	answeredAsks []peerasks.AnsweredCrossCheckedDocumentsAsk,
) peerjudgements.JudgedPeer {
	place := slices.IndexFunc(
		answeredAsks,
		func(answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk) bool {
			return answeredAsk.Answers(ask)
		},
	)
	if place < 0 {
		return peerjudgements.NoEvidenceFrom(ask.Peer.Hash)
	}
	answeredAsk := answeredAsks[place]

	return peerjudgements.JudgedPeerFrom(
		ask.Peer.Hash,
		answeredAsk.PeerVersion,
		judgementFromTheDocumentsListed(answeredAsk.DocumentsListedForTheWord, ask.Documents),
	)
}

func judgementFromTheDocumentsListed(
	documentsListed []yacymodel.URLHash,
	crossCheckedDocuments []yacymodel.URLHash,
) peerjudgements.Judgement {
	if len(documentsListed) == 0 {
		return peerjudgements.NoEvidence
	}
	for _, document := range documentsListed {
		if !slices.Contains(crossCheckedDocuments, document) {
			return peerjudgements.Ignored
		}
	}

	return peerjudgements.Honored
}
