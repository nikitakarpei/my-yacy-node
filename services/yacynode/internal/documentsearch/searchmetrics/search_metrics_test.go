package searchmetrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
)

func TestSearchMetricsCountSearchesPerOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := searchmetrics.NewSearchMetrics(registry)

	observer.ObserveSearchOutcome(searchmetrics.SearchServedWithResults)
	observer.ObserveSearchOutcome(searchmetrics.SearchServedWithResults)
	observer.ObserveSearchOutcome(searchmetrics.SearchIndexFailure)

	expected := `
# HELP documentsearch_searches_total Search requests answered, by how each one ended.
# TYPE documentsearch_searches_total counter
documentsearch_searches_total{outcome="index_failure"} 1
documentsearch_searches_total{outcome="served_with_results"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"documentsearch_searches_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestSearchMetricsSplitTermRingFractionsByPresence(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := searchmetrics.NewSearchMetrics(registry)

	observer.ObserveTermInIndex(0.01)
	observer.ObserveTermNotInIndex(0.2)
	observer.ObserveTermNotInIndex(0.3)

	if got := sampleCount(t, registry, "in_index"); got != 1 {
		t.Errorf("in_index samples = %v, want 1", got)
	}
	if got := sampleCount(t, registry, "not_in_index"); got != 2 {
		t.Errorf("not_in_index samples = %v, want 2", got)
	}
}

func sampleCount(t *testing.T, gatherer prometheus.Gatherer, presence string) uint64 {
	t.Helper()

	families, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	for _, family := range families {
		if family.GetName() != "documentsearch_query_term_ring_fraction" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "presence" && label.GetValue() == presence {
					return metric.GetHistogram().GetSampleCount()
				}
			}
		}
	}

	return 0
}

func TestSearchMetricsCountRequestsPerUnsupportedOption(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := searchmetrics.NewSearchMetrics(registry)

	observer.ObserveUnsupportedOptionRequested("prefer")
	observer.ObserveUnsupportedOptionRequested("prefer")

	expected := `
# HELP documentsearch_unsupported_options_requested_total Search options peers requested that this node accepts but ignores.
# TYPE documentsearch_unsupported_options_requested_total counter
documentsearch_unsupported_options_requested_total{option="prefer"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"documentsearch_unsupported_options_requested_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
