package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type crossCheckedDocumentsRound struct {
	asks          []peerasks.CrossCheckedDocumentsAsk
	answeredAsks  []peerasks.AnsweredCrossCheckedDocumentsAsk
	peerStandings peerjudgements.PeerStandings
}

func crossCheckedDocumentsRoundFrom(
	asks []peerasks.CrossCheckedDocumentsAsk,
	answeredAsks []peerasks.AnsweredCrossCheckedDocumentsAsk,
	peerStandings peerjudgements.PeerStandings,
) crossCheckedDocumentsRound {
	return crossCheckedDocumentsRound{
		asks:          asks,
		answeredAsks:  answeredAsks,
		peerStandings: peerStandings,
	}
}

func (round crossCheckedDocumentsRound) judgedPeers() []peerjudgements.JudgedPeer {
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
		judgementOfTheDocumentsHeld(answeredAsk.DocumentsHeldForTheWord, ask.Documents),
	)
}

func judgementOfTheDocumentsHeld(
	documentsHeld []yacymodel.URLHash,
	crossCheckedDocuments []yacymodel.URLHash,
) peerjudgements.Judgement {
	if len(documentsHeld) == 0 {
		return peerjudgements.NoEvidence
	}
	for _, document := range documentsHeld {
		if !slices.Contains(crossCheckedDocuments, document) {
			return peerjudgements.Ignored
		}
	}

	return peerjudgements.Honored
}

func (round crossCheckedDocumentsRound) documentsFoundByCrossCheckingPerQueryWord() documentsPerQueryWord {
	documentsFoundByCrossCheckingPerQueryWord := documentsPerQueryWord{}
	for _, answeredAsk := range round.answeredAsks {
		if documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] == nil {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word] = distinctDocuments{}
		}
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			documentsFoundByCrossCheckingPerQueryWord[answeredAsk.Ask.Word].add(document)
		}
	}

	return documentsFoundByCrossCheckingPerQueryWord
}
