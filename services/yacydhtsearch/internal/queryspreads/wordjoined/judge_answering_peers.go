package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func judgeAnsweringPeersIn(round crossCheckedDocumentsRound) []peerjudgements.JudgedPeer {
	judgedPeers := make([]peerjudgements.JudgedPeer, 0, len(round.answeredAsks))
	for _, answeredAsk := range round.answeredAsks {
		judgedPeers = append(judgedPeers, peerjudgements.JudgedPeerFrom(
			answeredAsk.Ask.Peer.Hash,
			answeredAsk.PeerVersion,
			judgementFromTheDocumentsListed(
				answeredAsk.DocumentsListedForTheWord, answeredAsk.Ask.Documents,
			),
		))
	}

	return judgedPeers
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
