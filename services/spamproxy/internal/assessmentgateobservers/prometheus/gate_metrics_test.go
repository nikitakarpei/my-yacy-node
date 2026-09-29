package prometheus_test

import (
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/assessmentgateobservers/prometheus"
)

func TestSlotWaitsAssessmentsAndPanicsAreCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address, err := canonicalurl.CanonicalURLOf("http://a.example/")
	if err != nil {
		t.Fatal(err)
	}

	metrics.SlotWaited(t.Context(), address, 20*time.Millisecond)
	metrics.AssessmentFinished(t.Context(), address, 8*time.Millisecond)
	metrics.AssessmentPanicked(t.Context(), address, "broken page")

	want := `
# HELP spamproxy_assessment_panics_total Assessments that panicked.
# TYPE spamproxy_assessment_panics_total counter
spamproxy_assessment_panics_total 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_assessment_panics_total"); err != nil {
		t.Fatal(err)
	}
	if series := testutil.CollectAndCount(
		registry,
		"spamproxy_assessment_slot_wait_duration_seconds",
		"spamproxy_assessment_duration_seconds",
	); series != 2 {
		t.Fatalf("histograms %d, want 2", series)
	}
}
