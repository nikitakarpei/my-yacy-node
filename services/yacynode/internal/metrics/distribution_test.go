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

func TestDistributionCountsPostingsGoneAndLedgerPrunes(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObservePostingsGone(1)
	observer.ObserveLedgerPrune(2)

	if got := testutil.ToFloat64(observer.postingsGone); got != 1 {
		t.Errorf("gone = %v, want 1", got)
	}
	if got := testutil.ToFloat64(observer.ledgerPrunes); got != 2 {
		t.Errorf("prunes = %v, want 2", got)
	}
}

func TestDistributionCountsUnreadReplication(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveReplicationUnread()

	if got := testutil.ToFloat64(observer.replicationUnread); got != 1 {
		t.Errorf("replication unread = %v, want 1", got)
	}
}

func TestDistributionTracksOldestDuePostingAge(t *testing.T) {
	observer := NewDistributionMetrics(prometheus.NewRegistry())

	observer.ObserveOldestDuePostingAge(90 * time.Second)

	if got := testutil.ToFloat64(observer.oldestDuePostingAge); got != 90 {
		t.Errorf("oldest due posting age = %v, want 90", got)
	}
}
