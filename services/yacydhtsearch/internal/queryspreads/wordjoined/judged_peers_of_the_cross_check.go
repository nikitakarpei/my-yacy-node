package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func judgedPeersOfTheCrossCheck(round crossCheckedDocumentsRound) []peerjudgements.JudgedPeer {
	judgedPeers := make([]peerjudgements.JudgedPeer, 0, len(round.asks))
	for _, ask := range round.asks {
		judgedPeers = append(judgedPeers, judgedPeerOf(ask, round.answeredAsks))
	}

	return judgedPeers
}

func judgedPeerOf(
	ask peerasks.CrossCheckedDocumentsAsk,
	answeredAsks []peerasks.AnsweredCrossCheckedDocumentsAsk,
) peerjudgements.JudgedPeer {
	answeredAsk, answered := answerToTheAskAmong(ask, answeredAsks).Get()
	if !answered {
		return peerjudgements.JudgedPeer{
			PeerAtVersion: peerjudgements.PeerAtVersion{Peer: ask.Peer.Hash},
			Judgement:     peerjudgements.NoEvidence,
		}
	}

	return peerjudgements.JudgedPeer{
		PeerAtVersion: peerjudgements.PeerAtVersion{
			Peer:    ask.Peer.Hash,
			Version: answeredAsk.PeerVersion,
		},
		Judgement: judgementOfTheDocumentsHeld(
			answeredAsk.DocumentsHeldForTheWord, ask.Documents,
		),
	}
}

func answerToTheAskAmong(
	ask peerasks.CrossCheckedDocumentsAsk,
	answeredAsks []peerasks.AnsweredCrossCheckedDocumentsAsk,
) yacymodel.Optional[peerasks.AnsweredCrossCheckedDocumentsAsk] {
	place := slices.IndexFunc(
		answeredAsks,
		func(answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk) bool {
			return answeredAsk.Ask.Peer.Hash == ask.Peer.Hash && answeredAsk.Ask.Word == ask.Word
		},
	)
	if place < 0 {
		return yacymodel.None[peerasks.AnsweredCrossCheckedDocumentsAsk]()
	}

	return yacymodel.Some(answeredAsks[place])
}

func judgementOfTheDocumentsHeld(
	documentsHeld []yacymodel.URLHash,
	namedDocuments []yacymodel.URLHash,
) peerjudgements.Judgement {
	if len(documentsHeld) == 0 {
		return peerjudgements.NoEvidence
	}
	for _, document := range documentsHeld {
		if !slices.Contains(namedDocuments, document) {
			return peerjudgements.Ignored
		}
	}

	return peerjudgements.Honored
}
