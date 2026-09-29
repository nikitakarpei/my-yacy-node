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
