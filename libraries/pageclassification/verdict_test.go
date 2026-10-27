package pageclassification_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

func TestEachVerdictHasItsName(t *testing.T) {
	t.Parallel()

	for verdict, wantedName := range map[pageclassification.Verdict]string{
		pageclassification.Unassessed: "unassessed",
		pageclassification.Negative:   "negative",
		pageclassification.Positive:   "positive",
	} {
		if verdict.String() != wantedName {
			t.Errorf("%d is named %q, want %q", verdict, verdict.String(), wantedName)
		}
	}
}

func TestACategoryWithoutAVerdictIsUnassessed(t *testing.T) {
	t.Parallel()

	verdicts := pageclassification.Verdicts{pageclassification.Spam: pageclassification.Negative}

	if verdict := verdicts["adult"]; verdict != pageclassification.Unassessed {
		t.Fatalf("verdict = %v, want unassessed", verdict)
	}
}
