package metrics_test

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestDistributionCountsOffersByResult(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePostingOffer("ok", 3)
	observer.ObservePostingOffer("ok", 2)
	observer.ObservePostingOffer("busy", 1)

	expected := `
# HELP rwidistribution_posting_offers_sent_total Posting offers sent to peers, by offer outcome.
# TYPE rwidistribution_posting_offers_sent_total counter
rwidistribution_posting_offers_sent_total{result="busy"} 1
rwidistribution_posting_offers_sent_total{result="ok"} 2
# HELP rwidistribution_postings_offered_total RWI postings offered to peers, by offer outcome.
# TYPE rwidistribution_postings_offered_total counter
rwidistribution_postings_offered_total{result="busy"} 1
rwidistribution_postings_offered_total{result="ok"} 5
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_posting_offers_sent_total",
		"rwidistribution_postings_offered_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsURLMetadataDeliveriesByResult(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveURLMetadataDelivery("accepted", 3)
	observer.ObserveURLMetadataDelivery("accepted", 2)
	observer.ObserveURLMetadataDelivery("deferred", 1)

	expected := `
# HELP rwidistribution_url_metadata_deliveries_total URL metadata deliveries sent to peers, by delivery outcome.
# TYPE rwidistribution_url_metadata_deliveries_total counter
rwidistribution_url_metadata_deliveries_total{result="accepted"} 2
rwidistribution_url_metadata_deliveries_total{result="deferred"} 1
# HELP rwidistribution_urls_delivered_total URLs delivered to peers as metadata, by delivery outcome.
# TYPE rwidistribution_urls_delivered_total counter
rwidistribution_urls_delivered_total{result="accepted"} 5
rwidistribution_urls_delivered_total{result="deferred"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_url_metadata_deliveries_total",
		"rwidistribution_urls_delivered_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsPostingsGoneAndStaleReplicasDropped(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePostingsGone(1)
	observer.ObserveStaleReplicasDropped(2)

	expected := `
# HELP rwidistribution_postings_gone_total Due postings evicted between the schedule read and the posting read.
# TYPE rwidistribution_postings_gone_total counter
rwidistribution_postings_gone_total 1
# HELP rwidistribution_stale_replicas_dropped_total Replicas dropped for peers no longer responsible.
# TYPE rwidistribution_stale_replicas_dropped_total counter
rwidistribution_stale_replicas_dropped_total 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_postings_gone_total",
		"rwidistribution_stale_replicas_dropped_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsSkippedCyclesByReason(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveCycleSkipped("too_few_reachable_peers")
	observer.ObserveCycleSkipped("too_few_reachable_peers")

	expected := `
# HELP rwidistribution_cycles_skipped_total Distribution cycles that ran no batch, by reason.
# TYPE rwidistribution_cycles_skipped_total counter
rwidistribution_cycles_skipped_total{reason="too_few_reachable_peers"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_cycles_skipped_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsAbortedBatchesByReason(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveBatchAborted("due_postings_unread")
	observer.ObserveBatchAborted("not_written")
	observer.ObserveBatchAborted("not_written")

	expected := `
# HELP rwidistribution_batches_aborted_total Distribution batches aborted before their postings were rescheduled, by reason.
# TYPE rwidistribution_batches_aborted_total counter
rwidistribution_batches_aborted_total{reason="due_postings_unread"} 1
rwidistribution_batches_aborted_total{reason="not_written"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_batches_aborted_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionTracksScheduledPostings(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveScheduledPostings(7)

	expected := `
# HELP rwidistribution_scheduled_postings Postings holding a due entry on the offer schedule.
# TYPE rwidistribution_scheduled_postings gauge
rwidistribution_scheduled_postings 7
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_scheduled_postings",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionTracksLongestOfferLateness(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveLongestOfferLateness(90 * time.Second)

	expected := `
# HELP rwidistribution_longest_offer_lateness_seconds Time the most overdue posting offer is past its scheduled time.
# TYPE rwidistribution_longest_offer_lateness_seconds gauge
rwidistribution_longest_offer_lateness_seconds 90
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"rwidistribution_longest_offer_lateness_seconds",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
