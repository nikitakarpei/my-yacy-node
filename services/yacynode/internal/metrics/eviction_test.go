package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestEvictionCountsSweptWork(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewEvictionMetrics(registry)

	observer.Observe(eviction.Result{URLsDeleted: 3, PostingsDeleted: 7})
	observer.Observe(eviction.Result{URLsDeleted: 2, PostingsDeleted: 1})

	expected := `
# HELP eviction_urls_evicted_total URLs purged by storage eviction.
# TYPE eviction_urls_evicted_total counter
eviction_urls_evicted_total 5
# HELP eviction_postings_evicted_total Postings purged by storage eviction.
# TYPE eviction_postings_evicted_total counter
eviction_postings_evicted_total 8
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"eviction_urls_evicted_total",
		"eviction_postings_evicted_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestEvictionCountsFailures(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewEvictionMetrics(registry)

	observer.ObserveFailure()
	observer.ObserveFailure()

	expected := `
# HELP eviction_failures_total Storage eviction sweeps that ended in error.
# TYPE eviction_failures_total counter
eviction_failures_total 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"eviction_failures_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
