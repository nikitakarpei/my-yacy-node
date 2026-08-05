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

	if got := testutil.ToFloat64(observer.postingOffers.WithLabelValues("ok")); got != 2 {
		t.Errorf("ok posting offers = %v, want 2", got)
	}
	if got := testutil.ToFloat64(observer.postingsOffered.WithLabelValues("ok")); got != 5 {
		t.Errorf("ok postings = %v, want 5", got)
	}
	if got := testutil.ToFloat64(observer.postingOffers.WithLabelValues("busy")); got != 1 {
		t.Errorf("busy posting offers = %v, want 1", got)
	}
}

func TestDistributionCountsURLMetadataDeliveriesByResult(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveURLMetadataDelivery("accepted", 3)
	observer.ObserveURLMetadataDelivery("accepted", 2)
	observer.ObserveURLMetadataDelivery("deferred", 1)

	if got := testutil.ToFloat64(
		observer.urlMetadataDeliveries.WithLabelValues("accepted"),
	); got != 2 {
		t.Errorf("accepted url metadata deliveries = %v, want 2", got)
	}
	if got := testutil.ToFloat64(observer.urlsDelivered.WithLabelValues("accepted")); got != 5 {
		t.Errorf("accepted urls delivered = %v, want 5", got)
	}
	if got := testutil.ToFloat64(
		observer.urlMetadataDeliveries.WithLabelValues("deferred"),
	); got != 1 {
		t.Errorf("deferred url metadata deliveries = %v, want 1", got)
	}
}

func TestDistributionCountsPostingsGoneAndStaleReplicasDropped(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObservePostingsGone(1)
	observer.ObserveStaleReplicasDropped(2)

	if got := testutil.ToFloat64(observer.postingsGone); got != 1 {
		t.Errorf("gone = %v, want 1", got)
	}
	if got := testutil.ToFloat64(observer.staleReplicasDropped); got != 2 {
		t.Errorf("staleReplicasDropped = %v, want 2", got)
	}
}

func TestDistributionCountsSkippedCyclesByReason(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveCycleSkipped("too_few_reachable_peers")
	observer.ObserveCycleSkipped("too_few_reachable_peers")

	if got := testutil.ToFloat64(
		observer.cyclesSkipped.WithLabelValues("too_few_reachable_peers"),
	); got != 2 {
		t.Errorf("cycles skipped for too few reachable peers = %v, want 2", got)
	}
}

func TestDistributionCountsAbortedBatchesByReason(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveBatchAborted("due_postings_unread")
	observer.ObserveBatchAborted("not_written")
	observer.ObserveBatchAborted("not_written")

	if got := testutil.ToFloat64(
		observer.batchesAborted.WithLabelValues("due_postings_unread"),
	); got != 1 {
		t.Errorf("batches aborted for unread due postings = %v, want 1", got)
	}
	if got := testutil.ToFloat64(
		observer.batchesAborted.WithLabelValues("not_written"),
	); got != 2 {
		t.Errorf("batches aborted for an unwritten batch = %v, want 2", got)
	}
}

func TestDistributionTracksScheduledPostings(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveScheduledPostings(7)

	if got := testutil.ToFloat64(observer.scheduledPostings); got != 7 {
		t.Errorf("scheduled postings = %v, want 7", got)
	}
}

func TestDistributionTracksLongestOfferLateness(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveLongestOfferLateness(90 * time.Second)

	if got := testutil.ToFloat64(observer.longestOfferLateness); got != 90 {
		t.Errorf("longest offer lateness = %v, want 90", got)
	}
}
