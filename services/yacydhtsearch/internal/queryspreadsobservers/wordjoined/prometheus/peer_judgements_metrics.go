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
	standingsPerStanding   map[peerjudgements.Standing]prometheusclient.Counter
	judgementsPerJudgement map[peerjudgements.Judgement]prometheusclient.Counter
}

func peerJudgementsMetricsRegisteredIn(
	registry prometheusclient.Registerer,
) peerJudgementsMetrics {
	standings := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_standings_total",
		Help: "How a peer stood on a question about it, by question and by standing.",
	}, []string{labelQuestion, labelStanding})
	judgements := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_judgements_total",
		Help: "How the answer of an asked peer was judged, by question and by judgement.",
	}, []string{labelQuestion, labelJudged})
	registry.MustRegister(standings, judgements)

	return peerJudgementsMetrics{
		standingsPerStanding:   standingsPerStandingFrom(standings),
		judgementsPerJudgement: judgementsPerJudgementFrom(judgements),
	}
}

func standingsPerStandingFrom(
	standings *prometheusclient.CounterVec,
) map[peerjudgements.Standing]prometheusclient.Counter {
	//exhaustive:enforce
	return map[peerjudgements.Standing]prometheusclient.Counter{
		peerjudgements.Honoring:       standingsCountedAs(standings, peerjudgements.Honoring),
		peerjudgements.Ignoring:       standingsCountedAs(standings, peerjudgements.Ignoring),
		peerjudgements.NeverJudged:    standingsCountedAs(standings, peerjudgements.NeverJudged),
		peerjudgements.VersionChanged: standingsCountedAs(standings, peerjudgements.VersionChanged),
		peerjudgements.IntervalPassed: standingsCountedAs(standings, peerjudgements.IntervalPassed),
	}
}

func standingsCountedAs(
	standings *prometheusclient.CounterVec,
	standing peerjudgements.Standing,
) prometheusclient.Counter {
	return standings.WithLabelValues(
		string(wordjoined.ListsOnlyTheCrossCheckedDocuments), string(standing),
	)
}

func judgementsPerJudgementFrom(
	judgements *prometheusclient.CounterVec,
) map[peerjudgements.Judgement]prometheusclient.Counter {
	//exhaustive:enforce
	return map[peerjudgements.Judgement]prometheusclient.Counter{
		peerjudgements.Honored:    judgementsCountedAs(judgements, peerjudgements.Honored),
		peerjudgements.Ignored:    judgementsCountedAs(judgements, peerjudgements.Ignored),
		peerjudgements.NoEvidence: judgementsCountedAs(judgements, peerjudgements.NoEvidence),
	}
}

func judgementsCountedAs(
	judgements *prometheusclient.CounterVec,
	judgement peerjudgements.Judgement,
) prometheusclient.Counter {
	return judgements.WithLabelValues(
		string(wordjoined.ListsOnlyTheCrossCheckedDocuments), string(judgement),
	)
}

func (m peerJudgementsMetrics) countStandingsAndJudgements(
	crossCheckedDocumentsRound wordjoined.PerformedCrossCheckedDocumentsRound,
) {
	for _, peerStanding := range crossCheckedDocumentsRound.PeerStandings {
		m.standingsPerStanding[peerStanding.Standing].Inc()
	}
	for _, judgedPeer := range crossCheckedDocumentsRound.JudgedPeers {
		m.judgementsPerJudgement[judgedPeer.Judgement].Inc()
	}
}
