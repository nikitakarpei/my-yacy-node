package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDistributionCountsOffersByResult(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObservePostingOffer("ok", 3)
	observer.ObservePostingOffer("ok", 2)
	observer.ObservePostingOffer("busy", 1)

	if got := testutil.ToFloat64(observer.offerRequests.WithLabelValues("ok")); got != 2 {
		t.Errorf("ok offer requests = %v, want 2", got)
	}
	if got := testutil.ToFloat64(observer.postingsOffered.WithLabelValues("ok")); got != 5 {
		t.Errorf("ok postings = %v, want 5", got)
	}
	if got := testutil.ToFloat64(observer.offerRequests.WithLabelValues("busy")); got != 1 {
		t.Errorf("busy offer requests = %v, want 1", got)
	}
}

func TestDistributionCountsPostingsDueAndGoneAndLedgerPrunes(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObservePostingsDue(4)
	observer.ObservePostingsDue(6)
	observer.ObservePostingsGone(1)
	observer.ObserveLedgerPrune(2)

	if got := testutil.ToFloat64(observer.postingsDue); got != 10 {
		t.Errorf("due = %v, want 10", got)
	}
	if got := testutil.ToFloat64(observer.postingsGone); got != 1 {
		t.Errorf("gone = %v, want 1", got)
	}
	if got := testutil.ToFloat64(observer.ledgerPrunes); got != 2 {
		t.Errorf("prunes = %v, want 2", got)
	}
}

func TestDistributionTracksOldestDuePostingAge(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveOldestDuePostingAge(90 * time.Second)

	if got := testutil.ToFloat64(observer.oldestDuePostingAge); got != 90 {
		t.Errorf("oldest due posting age = %v, want 90", got)
	}
}
