// Package prometheus counts how the peers stand on each form of an ask, and
// how the answer of each asked peer was judged.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

const (
	labelForm     = "form"
	labelStanding = "standing"
	labelJudged   = "judged"
)

type JudgementsMetrics struct {
	standingsPerForm  map[peerjudgements.Form]standingsOfOneForm
	judgementsPerForm map[peerjudgements.Form]judgementsOfOneForm
}

type standingsOfOneForm map[peerjudgements.Standing]prometheusclient.Counter

type judgementsOfOneForm map[peerjudgements.Judgement]prometheusclient.Counter

func New(registry prometheusclient.Registerer) *JudgementsMetrics {
	standings := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_standings_total",
		Help: "How a peer stood on a form of an ask, by form and by standing.",
	}, []string{labelForm, labelStanding})
	judgements := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_peer_judgements_total",
		Help: "How the answer of an asked peer was judged, by form and by judgement.",
	}, []string{labelForm, labelJudged})
	registry.MustRegister(standings, judgements)

	//exhaustive:enforce
	standingsPerForm := map[peerjudgements.Form]standingsOfOneForm{
		peerjudgements.NamedDocuments: standingsOfOneFormFrom(
			standings, peerjudgements.NamedDocuments,
		),
	}
	//exhaustive:enforce
	judgementsPerForm := map[peerjudgements.Form]judgementsOfOneForm{
		peerjudgements.NamedDocuments: judgementsOfOneFormFrom(
			judgements, peerjudgements.NamedDocuments,
		),
	}

	return &JudgementsMetrics{
		standingsPerForm:  standingsPerForm,
		judgementsPerForm: judgementsPerForm,
	}
}

func standingsOfOneFormFrom(
	standings *prometheusclient.CounterVec,
	form peerjudgements.Form,
) standingsOfOneForm {
	//exhaustive:enforce
	return standingsOfOneForm{
		peerjudgements.Honoring: standingsCountedAs(standings, form, peerjudgements.Honoring),
		peerjudgements.Ignoring: standingsCountedAs(standings, form, peerjudgements.Ignoring),
		peerjudgements.NeverJudged: standingsCountedAs(
			standings,
			form,
			peerjudgements.NeverJudged,
		),
		peerjudgements.VersionChanged: standingsCountedAs(
			standings,
			form,
			peerjudgements.VersionChanged,
		),
		peerjudgements.IntervalPassed: standingsCountedAs(
			standings,
			form,
			peerjudgements.IntervalPassed,
		),
	}
}

func standingsCountedAs(
	standings *prometheusclient.CounterVec,
	form peerjudgements.Form,
	standing peerjudgements.Standing,
) prometheusclient.Counter {
	return standings.WithLabelValues(string(form), string(standing))
}

func judgementsOfOneFormFrom(
	judgements *prometheusclient.CounterVec,
	form peerjudgements.Form,
) judgementsOfOneForm {
	//exhaustive:enforce
	return judgementsOfOneForm{
		peerjudgements.Honored:    judgementsCountedAs(judgements, form, peerjudgements.Honored),
		peerjudgements.Ignored:    judgementsCountedAs(judgements, form, peerjudgements.Ignored),
		peerjudgements.NoEvidence: judgementsCountedAs(judgements, form, peerjudgements.NoEvidence),
	}
}

func judgementsCountedAs(
	judgements *prometheusclient.CounterVec,
	form peerjudgements.Form,
	judgement peerjudgements.Judgement,
) prometheusclient.Counter {
	return judgements.WithLabelValues(string(form), string(judgement))
}

func (m *JudgementsMetrics) PeerStood(
	_ context.Context,
	form peerjudgements.Form,
	standing peerjudgements.PeerStanding,
) {
	m.standingsPerForm[form][standing.Standing].Inc()
}

func (m *JudgementsMetrics) PeerJudged(
	_ context.Context,
	form peerjudgements.Form,
	judgedPeer peerjudgements.JudgedPeer,
) {
	m.judgementsPerForm[form][judgedPeer.Judgement].Inc()
}
