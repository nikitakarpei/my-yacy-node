package searchmetrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestSearchMetricsCountSearchesPerOutcome(t *testing.T) {
	observer := NewSearchMetrics(prometheus.NewRegistry())

	observer.ObserveSearchOutcome(SearchServedWithResults)
	observer.ObserveSearchOutcome(SearchServedWithResults)
	observer.ObserveSearchOutcome(SearchIndexFailure)

	if got := testutil.ToFloat64(
		observer.searchesPerOutcome.WithLabelValues(string(SearchServedWithResults)),
	); got != 2 {
		t.Errorf("served_with_results = %v, want 2", got)
	}
	if got := testutil.ToFloat64(
		observer.searchesPerOutcome.WithLabelValues(string(SearchIndexFailure)),
	); got != 1 {
		t.Errorf("index_failure = %v, want 1", got)
	}
}

func TestSearchMetricsSplitTermRingFractionsByPresence(t *testing.T) {
	observer := NewSearchMetrics(prometheus.NewRegistry())

	observer.ObserveTermInIndex(0.01)
	observer.ObserveTermNotInIndex(0.2)
	observer.ObserveTermNotInIndex(0.3)

	if got := sampleCount(t, observer, termInIndex); got != 1 {
		t.Errorf("in_index samples = %v, want 1", got)
	}
	if got := sampleCount(t, observer, termNotInIndex); got != 2 {
		t.Errorf("not_in_index samples = %v, want 2", got)
	}
}

func sampleCount(t *testing.T, observer *SearchMetrics, presence string) uint64 {
	t.Helper()

	histogram, err := observer.termRingFractionPerPresence.GetMetricWithLabelValues(presence)
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues: %v", err)
	}
	var collected dto.Metric
	if err := histogram.(prometheus.Histogram).Write(&collected); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return collected.GetHistogram().GetSampleCount()
}

func TestSearchMetricsCountRequestsPerUnsupportedOption(t *testing.T) {
	observer := NewSearchMetrics(prometheus.NewRegistry())

	observer.ObserveUnsupportedOptionRequested("prefer")
	observer.ObserveUnsupportedOptionRequested("prefer")

	if got := testutil.ToFloat64(
		observer.requestsPerUnsupportedOption.WithLabelValues("prefer"),
	); got != 2 {
		t.Errorf("prefer requests = %v, want 2", got)
	}
}
