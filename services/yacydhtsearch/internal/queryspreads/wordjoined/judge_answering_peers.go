package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

func judgeAnsweringPeersIn(round crossCheckedDocumentsRound) []peerjudgements.JudgedPeer {
	judgedPeers := make([]peerjudgements.JudgedPeer, 0, len(round.answeredAsks))
	for _, answeredAsk := range round.answeredAsks {
		judgedPeers = append(judgedPeers, peerjudgements.JudgedPeerFrom(
			answeredAsk.Ask.Peer.Hash,
			answeredAsk.Ask.Peer.Version,
			judgementFrom(answeredAsk),
		))
	}

	return judgedPeers
}

func judgementFrom(answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk) peerjudgements.Judgement {
	if len(answeredAsk.DocumentsListedForTheWord) == 0 {
		return peerjudgements.NoEvidence
	}
	if !answeredAsk.ListsOnlyTheDocumentsAsked() {
		return peerjudgements.Ignored
	}

	return peerjudgements.Honored
}
