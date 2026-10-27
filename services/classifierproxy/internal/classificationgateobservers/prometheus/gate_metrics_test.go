package prometheus_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/classificationgateobservers/prometheus"
)

func TestSlotWaitsClassificationsAndPanicsAreCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address := addressOf(t)

	metrics.SlotWaited(t.Context(), address, 20*time.Millisecond)
	metrics.ClassificationFinished(t.Context(), address, 1000, 8*time.Millisecond)
	metrics.ClassificationPanicked(t.Context(), address, "broken page")

	want := `
# HELP classifierproxy_classification_panics_total Classifications that panicked.
# TYPE classifierproxy_classification_panics_total counter
classifierproxy_classification_panics_total 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"classifierproxy_classification_panics_total"); err != nil {
		t.Fatal(err)
	}
	if series := testutil.CollectAndCount(
		registry,
		"classifierproxy_classification_slot_wait_duration_seconds",
		"classifierproxy_classification_duration_seconds",
	); series != 2 {
		t.Fatalf("histograms %d, want 2", series)
	}
}

func TestClassificationTimesAreCountedByPageSize(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address := addressOf(t)

	for _, pageSize := range []int{16 << 10, 16<<10 + 1, 256 << 10, 1 << 20, 1<<20 + 1} {
		metrics.ClassificationFinished(t.Context(), address, pageSize, 8*time.Millisecond)
	}

	want := []string{"over_1MiB", "up_to_16KiB", "up_to_1MiB", "up_to_256KiB", "up_to_64KiB"}
	if got := pageSizeRangesIn(t, registry); !slices.Equal(got, want) {
		t.Fatalf("page size ranges %v, want %v", got, want)
	}
}

func pageSizeRangesIn(t *testing.T, registry *prometheusclient.Registry) []string {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var sizeRanges []string
	for _, family := range families {
		if family.GetName() != "classifierproxy_classification_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				sizeRanges = append(sizeRanges, label.GetValue())
			}
		}
	}
	slices.Sort(sizeRanges)
	return sizeRanges
}

func addressOf(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	address, err := canonicalurl.CanonicalURLOf("http://a.example/")
	if err != nil {
		t.Fatal(err)
	}
	return address
}
