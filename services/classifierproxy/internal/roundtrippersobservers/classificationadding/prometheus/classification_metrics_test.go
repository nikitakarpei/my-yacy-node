package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/classificationadding"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippersobservers/classificationadding/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

func TestVerdictsSkippedClassificationsRefusalsAndFailuresAreCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address, err := canonicalurl.CanonicalURLOf("http://a.example/")
	if err != nil {
		t.Fatal(err)
	}

	metrics.PageClassified(
		t.Context(),
		address,
		pageclassification.Classification{
			pageclassification.Spam: {Score: 0.9, Threshold: 0.8},
			"adult":                 {Score: 0.1, Threshold: 0.8},
		},
	)
	metrics.ClassificationSkipped(t.Context(), address, classificationadding.NotHTML)
	metrics.PageRefused(t.Context(), address, classificationadding.SlotWaitDeadline)
	metrics.BodyPrefixReadFailed(t.Context(), address, errors.New("egress proxy down"))
	metrics.ReadingSlotWaited(t.Context(), address, 0)

	want := `
# HELP classifierproxy_body_prefix_read_failures_total Pages whose body failed before the part that the proxy classifies.
# TYPE classifierproxy_body_prefix_read_failures_total counter
classifierproxy_body_prefix_read_failures_total 1
# HELP classifierproxy_skipped_classifications_total Upstream responses relayed without a classification.
# TYPE classifierproxy_skipped_classifications_total counter
classifierproxy_skipped_classifications_total{reason="not_html"} 1
# HELP classifierproxy_refused_pages_total Pages that the proxy answers with its own error status, by reason.
# TYPE classifierproxy_refused_pages_total counter
classifierproxy_refused_pages_total{reason="slot_wait_deadline"} 1
# HELP classifierproxy_classified_pages_total Classified pages.
# TYPE classifierproxy_classified_pages_total counter
classifierproxy_classified_pages_total 1
# HELP classifierproxy_verdicts_total Verdicts on the classified pages, by category and verdict.
# TYPE classifierproxy_verdicts_total counter
classifierproxy_verdicts_total{category="adult",verdict="negative"} 1
classifierproxy_verdicts_total{category="spam",verdict="positive"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"classifierproxy_skipped_classifications_total", "classifierproxy_refused_pages_total",
		"classifierproxy_body_prefix_read_failures_total", "classifierproxy_classified_pages_total",
		"classifierproxy_verdicts_total"); err != nil {
		t.Fatal(err)
	}
	if series := testutil.CollectAndCount(
		registry,
		"classifierproxy_scores",
		"classifierproxy_reading_slot_wait_duration_seconds",
	); series != 3 {
		t.Fatalf("histograms %d, want 3", series)
	}
}
