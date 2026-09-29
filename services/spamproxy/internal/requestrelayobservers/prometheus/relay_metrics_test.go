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
	)
	metrics.AssessmentSkipped(t.Context(), address, requestrelay.NotHTML)
	metrics.RequestRefused(t.Context(), address, requestrelay.SlotWaitDeadline)
	metrics.UpstreamResponseFailed(t.Context(), address, requestrelay.NoResponse,
		errors.New("egress proxy down"))
	metrics.ResponseLeftIncomplete(t.Context(), address, requestrelay.RelayIdleTimeout,
		errors.New("upstream response stalled"))
	metrics.ClientClosedRequest(t.Context(), address)
	metrics.ReadingSlotWaited(t.Context(), address, 0)

	want := `
# HELP spamproxy_upstream_response_failures_total Upstream responses that failed before the response headers, by cause.
# TYPE spamproxy_upstream_response_failures_total counter
spamproxy_upstream_response_failures_total{cause="no_response"} 1
# HELP spamproxy_incomplete_responses_total Responses that ended before the whole body, by cause.
# TYPE spamproxy_incomplete_responses_total counter
spamproxy_incomplete_responses_total{cause="relay_idle_timeout"} 1
# HELP spamproxy_client_closed_requests_total Requests that the client closed before the response headers.
# TYPE spamproxy_client_closed_requests_total counter
spamproxy_client_closed_requests_total 1
# HELP spamproxy_skipped_assessments_total Upstream responses relayed without an assessment.
# TYPE spamproxy_skipped_assessments_total counter
spamproxy_skipped_assessments_total{reason="not_html"} 1
# HELP spamproxy_refused_requests_total Requests that the proxy answers with its own error status.
# TYPE spamproxy_refused_requests_total counter
spamproxy_refused_requests_total{reason="slot_wait_deadline"} 1
# HELP spamproxy_verdicts_total Assessed pages by verdict.
# TYPE spamproxy_verdicts_total counter
spamproxy_verdicts_total{verdict="spam"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_skipped_assessments_total", "spamproxy_refused_requests_total",
		"spamproxy_upstream_response_failures_total", "spamproxy_incomplete_responses_total",
		"spamproxy_client_closed_requests_total",
		"spamproxy_verdicts_total"); err != nil {
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
