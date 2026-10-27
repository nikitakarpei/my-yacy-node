package pageclassification_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

func TestAScoreAboveTheThresholdIsPositive(t *testing.T) {
	t.Parallel()

	assessment := pageclassification.Assessment{Score: 0.81, Threshold: 0.8}

	if assessment.Verdict() != pageclassification.Positive {
		t.Fatalf("verdict = %v, want positive", assessment.Verdict())
	}
}

func TestAScoreAtTheThresholdIsNegative(t *testing.T) {
	t.Parallel()

	assessment := pageclassification.Assessment{Score: 0.8, Threshold: 0.8}

	if assessment.Verdict() != pageclassification.Negative {
		t.Fatalf("verdict = %v, want negative", assessment.Verdict())
	}
}
