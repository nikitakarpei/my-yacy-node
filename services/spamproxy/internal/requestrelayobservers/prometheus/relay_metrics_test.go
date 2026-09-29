package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelayobservers/prometheus"
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
		0,
	)
	metrics.AssessmentSkipped(t.Context(), address, requestrelay.NotHTML)
	metrics.RequestRefused(t.Context(), address, requestrelay.SlotWaitDeadline)
	metrics.AnswerReadingFailed(t.Context(), address, errors.New("egress proxy down"))
	metrics.ReplyCutShort(t.Context(), address, errors.New("egress stopped"))
	metrics.ClientLeft(t.Context(), address)

	want := `
# HELP spamproxy_answer_reading_failures_total Answers of the egress proxy that failed before the response headers.
# TYPE spamproxy_answer_reading_failures_total counter
spamproxy_answer_reading_failures_total 1
# HELP spamproxy_cut_short_replies_total Replies that ended before the whole body.
# TYPE spamproxy_cut_short_replies_total counter
spamproxy_cut_short_replies_total 1
# HELP spamproxy_departed_clients_total Clients that left before the reply ended.
# TYPE spamproxy_departed_clients_total counter
spamproxy_departed_clients_total 1
# HELP spamproxy_skipped_assessments_total Answers relayed without an assessment.
# TYPE spamproxy_skipped_assessments_total counter
spamproxy_skipped_assessments_total{reason="not_html"} 1
# HELP spamproxy_refusals_total Answers refused with an error status.
# TYPE spamproxy_refusals_total counter
spamproxy_refusals_total{reason="slot_wait_deadline"} 1
# HELP spamproxy_verdicts_total Assessed pages by verdict.
# TYPE spamproxy_verdicts_total counter
spamproxy_verdicts_total{verdict="spam"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_skipped_assessments_total", "spamproxy_refusals_total",
		"spamproxy_answer_reading_failures_total", "spamproxy_cut_short_replies_total",
		"spamproxy_departed_clients_total",
		"spamproxy_verdicts_total"); err != nil {
		t.Fatal(err)
	}
	if series := testutil.CollectAndCount(
		registry,
		"spamproxy_scores",
		"spamproxy_assessment_duration_seconds",
	); series != 2 {
		t.Fatalf("histograms %d, want 2", series)
	}
}
