package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

func crossCheckJudgementsIn(round crossCheckRound) []peerjudgements.JudgedPeer {
	judgedPeers := make([]peerjudgements.JudgedPeer, 0, len(round.asks))
	for _, ask := range round.asks {
		judgedPeers = append(judgedPeers, judgeAskedPeer(ask, round.answeredAsks))
	}

	return judgedPeers
}

func judgeAskedPeer(
	ask peerasks.SearchDocumentsAsk,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) peerjudgements.JudgedPeer {
	place := slices.IndexFunc(
		answeredAsks,
		func(answeredAsk peerasks.AnsweredSearchDocumentsAsk) bool {
			return answeredAsk.Ask.Peer.Hash == ask.Peer.Hash && answeredAsk.Ask.Word == ask.Word
		},
	)
	if place < 0 {
		return peerjudgements.NoEvidenceFrom(ask.Peer.Hash)
	}
	answeredAsk := answeredAsks[place]

	return peerjudgements.JudgedPeerFrom(
		ask.Peer.Hash,
		answeredAsk.PeerVersion,
		crossCheckJudgementOf(answeredAsk),
	)
}

func crossCheckJudgementOf(
	answeredAsk peerasks.AnsweredSearchDocumentsAsk,
) peerjudgements.Judgement {
	if len(answeredAsk.Abstract) == 0 {
		return peerjudgements.NoEvidence
	}
	if answeredAsk.IgnoredTheDocumentsToMatch() {
		return peerjudgements.Ignored
	}

	return peerjudgements.Honored
}
