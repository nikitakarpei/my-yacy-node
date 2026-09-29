package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/pagerelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

func TestVerdictsNonPagesRefusalsAndFailuresAreCounted(t *testing.T) {
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
	metrics.NonPageRelayed(t.Context(), address, pagerelay.NotHTML)
	metrics.AnswerRefused(t.Context(), address, pagerelay.TooManyAtOnce)
	metrics.RelayFailed(t.Context(), address, errors.New("egress proxy down"))

	want := `
# HELP spamproxy_non_pages_total Answers relayed without a verdict because they are not pages.
# TYPE spamproxy_non_pages_total counter
spamproxy_non_pages_total{reason="not_html"} 1
# HELP spamproxy_refusals_total Answers refused with an error status.
# TYPE spamproxy_refusals_total counter
spamproxy_refusals_total{reason="too_many_at_once"} 1
# HELP spamproxy_relay_failures_total Relays that failed.
# TYPE spamproxy_relay_failures_total counter
spamproxy_relay_failures_total 1
# HELP spamproxy_verdicts_total Assessed pages by verdict.
# TYPE spamproxy_verdicts_total counter
spamproxy_verdicts_total{verdict="spam"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_non_pages_total", "spamproxy_refusals_total",
		"spamproxy_relay_failures_total", "spamproxy_verdicts_total"); err != nil {
		t.Fatal(err)
	}
	if series := testutil.CollectAndCount(
		registry,
		"spamproxy_scores",
		"spamproxy_assessment_seconds",
	); series != 2 {
		t.Fatalf("histograms %d, want 2", series)
	}
}
