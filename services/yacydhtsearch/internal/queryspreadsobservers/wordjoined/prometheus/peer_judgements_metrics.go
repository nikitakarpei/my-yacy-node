package prometheus

import (
	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
)

const (
	labelQuestion = "question"
	labelStanding = "standing"
	labelJudged   = "judged"
)

type peerJudgementsMetrics struct {
	standingCounters  map[peerjudgements.Standing]prometheusclient.Counter
	judgementCounters map[peerjudgements.Judgement]prometheusclient.Counter
}

func peerJudgementsMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) peerJudgementsMetrics {
	peerStandings := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_peer_standings_total",
		Help: "Peers the spread considered for the cross-check, by the question and their " +
			"standing on it.",
	}, []string{labelQuestion, labelStanding})
	peerJudgements := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_peer_judgements_total",
		Help: "Answers of the asked peers, by the question and how each was judged.",
	}, []string{labelQuestion, labelJudged})
	registry.MustRegister(peerStandings, peerJudgements)

	question := prometheusclient.Labels{
		labelQuestion: string(wordjoined.ListsOnlyTheCrossCheckedDocuments),
	}
	standings := peerStandings.MustCurryWith(question)
	judgements := peerJudgements.MustCurryWith(question)

	return peerJudgementsMetrics{
		//exhaustive:enforce
		standingCounters: map[peerjudgements.Standing]prometheusclient.Counter{
			peerjudgements.Honoring: standings.WithLabelValues(
				string(peerjudgements.Honoring),
			),
			peerjudgements.Ignoring: standings.WithLabelValues(
				string(peerjudgements.Ignoring),
			),
			peerjudgements.NeverJudged: standings.WithLabelValues(
				string(peerjudgements.NeverJudged),
			),
			peerjudgements.VersionChanged: standings.WithLabelValues(
				string(peerjudgements.VersionChanged),
			),
			peerjudgements.IntervalPassed: standings.WithLabelValues(
				string(peerjudgements.IntervalPassed),
			),
		},
		//exhaustive:enforce
		judgementCounters: map[peerjudgements.Judgement]prometheusclient.Counter{
			peerjudgements.Honored: judgements.WithLabelValues(string(peerjudgements.Honored)),
			peerjudgements.Ignored: judgements.WithLabelValues(string(peerjudgements.Ignored)),
			peerjudgements.NoEvidence: judgements.WithLabelValues(
				string(peerjudgements.NoEvidence),
			),
		},
	}
}

func (m peerJudgementsMetrics) countStandingsAndJudgements(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
) {
	for _, peerStanding := range crossCheckedDocumentsRound.PeerStandings {
		m.standingCounters[peerStanding.Standing].Inc()
	}
	for _, judgedPeer := range crossCheckedDocumentsRound.JudgedPeers {
		m.judgementCounters[judgedPeer.Judgement].Inc()
	}
}
