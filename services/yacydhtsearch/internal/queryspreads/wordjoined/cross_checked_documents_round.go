package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

type crossCheckedDocumentsRound struct {
	asks          []peerasks.CrossCheckedDocumentsAsk
	answeredAsks  []peerasks.AnsweredCrossCheckedDocumentsAsk
	peerStandings []peerjudgements.PeerStanding
	judgedPeers   []peerjudgements.JudgedPeer
}

func crossCheckedDocumentsRoundFrom(
	asks []peerasks.CrossCheckedDocumentsAsk,
	answeredAsks []peerasks.AnsweredCrossCheckedDocumentsAsk,
	peerStandings []peerjudgements.PeerStanding,
) crossCheckedDocumentsRound {
	return crossCheckedDocumentsRound{
		asks:          asks,
		answeredAsks:  answeredAsks,
		peerStandings: peerStandings,
		judgedPeers:   judgedPeersOfTheCrossCheck(asks, answeredAsks),
	}
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
