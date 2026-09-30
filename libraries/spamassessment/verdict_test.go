package spamassessment_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

func TestEachVerdictHasItsName(t *testing.T) {
	t.Parallel()

	for verdict, wantedName := range map[spamassessment.Verdict]string{
		spamassessment.Unassessed: "unassessed",
		spamassessment.Clean:      "clean",
		spamassessment.Spam:       "spam",
	} {
		if verdict.String() != wantedName {
			t.Errorf("%d is named %q, want %q", verdict, verdict.String(), wantedName)
		}
	}
}
