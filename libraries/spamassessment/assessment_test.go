package spamassessment_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

func TestAScoreAboveTheThresholdIsSpam(t *testing.T) {
	t.Parallel()

	assessment := spamassessment.Assessment{Score: 0.81, Threshold: 0.8}

	if assessment.Verdict() != spamassessment.Spam {
		t.Fatalf("verdict = %v, want spam", assessment.Verdict())
	}
}

func TestAScoreAtTheThresholdIsClean(t *testing.T) {
	t.Parallel()

	assessment := spamassessment.Assessment{Score: 0.8, Threshold: 0.8}

	if assessment.Verdict() != spamassessment.Clean {
		t.Fatalf("verdict = %v, want clean", assessment.Verdict())
	}
}
