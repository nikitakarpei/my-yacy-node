package httpheader_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
)

func TestTheTokenOfTheValueIsTheVerdict(t *testing.T) {
	t.Parallel()

	for spamAssessmentValue, wantedVerdict := range map[string]spamassessment.Verdict{
		`spam;score=0.935;threshold=0.8;model="2026-09"`:  spamassessment.Spam,
		`clean;score=0.007;threshold=0.8;model="2026-09"`: spamassessment.Clean,
		"clean":          spamassessment.Clean,
		"":               spamassessment.Unassessed,
		"SPAM;score=0.9": spamassessment.Unassessed,
		`"spam"`:         spamassessment.Unassessed,
		"spammy":         spamassessment.Unassessed,
	} {
		if verdict := httpheader.VerdictFrom(spamAssessmentValue); verdict != wantedVerdict {
			t.Errorf("%q yields %v, want %v", spamAssessmentValue, verdict, wantedVerdict)
		}
	}
}

func TestTheValueCarriesTheVerdictScoreThresholdAndModel(t *testing.T) {
	t.Parallel()

	assessment := spamassessment.Assessment{Score: 0.93549, Threshold: 0.8, ModelVersion: "2026-09"}

	if value, want := httpheader.ValueOf(
		assessment,
	), `spam;score=0.935;threshold=0.8;model="2026-09"`; value != want {
		t.Fatalf("value = %s, want %s", value, want)
	}
}

func TestTheVerdictReadFromAValueIsTheVerdictWritten(t *testing.T) {
	t.Parallel()

	for _, assessment := range []spamassessment.Assessment{
		{Score: 0.9, Threshold: 0.8, ModelVersion: "2026-09"},
		{Score: 0.1, Threshold: 0.8, ModelVersion: "2026-09"},
	} {
		if verdict := httpheader.VerdictFrom(
			httpheader.ValueOf(assessment),
		); verdict != assessment.Verdict() {
			t.Errorf("%+v reads back as %v", assessment, verdict)
		}
	}
}
