package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/verdictadding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/verdictadding/prometheus"
)

func TestVerdictsSkippedAssessmentsRefusalsAndFailuresAreCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address, err := canonicalurl.CanonicalURLOf("http://a.example/")
	if err != nil {
		t.Fatal(err)
	}

	metrics.PageAssessed(
		t.Context(),
		address,
		spamassessment.Assessment{Score: 0.9, Threshold: 0.8},
	)
	metrics.AssessmentSkipped(t.Context(), address, verdictadding.NotHTML)
	metrics.PageRefused(t.Context(), address, verdictadding.SlotWaitDeadline)
	metrics.BodyPrefixReadFailed(t.Context(), address, errors.New("egress proxy down"))
	metrics.ReadingSlotWaited(t.Context(), address, 0)

	want := `
# HELP spamproxy_body_prefix_read_failures_total Pages whose body failed before the part that the proxy assesses.
# TYPE spamproxy_body_prefix_read_failures_total counter
spamproxy_body_prefix_read_failures_total 1
# HELP spamproxy_skipped_assessments_total Upstream responses relayed without an assessment.
# TYPE spamproxy_skipped_assessments_total counter
spamproxy_skipped_assessments_total{reason="not_html"} 1
# HELP spamproxy_refused_pages_total Pages that the proxy answers with its own error status, by reason.
# TYPE spamproxy_refused_pages_total counter
spamproxy_refused_pages_total{reason="slot_wait_deadline"} 1
# HELP spamproxy_verdicts_total Assessed pages by verdict.
# TYPE spamproxy_verdicts_total counter
spamproxy_verdicts_total{verdict="spam"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_skipped_assessments_total", "spamproxy_refused_pages_total",
		"spamproxy_body_prefix_read_failures_total", "spamproxy_verdicts_total"); err != nil {
		t.Fatal(err)
	}
	if series := testutil.CollectAndCount(
		registry,
		"spamproxy_scores",
		"spamproxy_reading_slot_wait_duration_seconds",
	); series != 2 {
		t.Fatalf("histograms %d, want 2", series)
	}
}
