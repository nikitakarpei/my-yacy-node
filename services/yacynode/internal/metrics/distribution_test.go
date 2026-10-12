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
# HELP yacynode_rwidistribution_posting_offers_sent_total Posting offers sent to peers, by offer outcome.
# TYPE yacynode_rwidistribution_posting_offers_sent_total counter
yacynode_rwidistribution_posting_offers_sent_total{result="busy"} 1
yacynode_rwidistribution_posting_offers_sent_total{result="ok"} 2
# HELP yacynode_rwidistribution_postings_offered_total RWI postings offered to peers, by offer outcome.
# TYPE yacynode_rwidistribution_postings_offered_total counter
yacynode_rwidistribution_postings_offered_total{result="busy"} 1
yacynode_rwidistribution_postings_offered_total{result="ok"} 5
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_posting_offers_sent_total",
		"yacynode_rwidistribution_postings_offered_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsURLsUnknownToUs(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveURLsUnknownToUs(3)
	observer.ObserveURLsUnknownToUs(2)

	expected := `
# HELP yacynode_rwidistribution_urls_unknown_to_us_total URLs a peer asked for whose metadata this node does not hold, so no delivery carried them.
# TYPE yacynode_rwidistribution_urls_unknown_to_us_total counter
yacynode_rwidistribution_urls_unknown_to_us_total 5
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_urls_unknown_to_us_total",
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
# HELP yacynode_rwidistribution_url_metadata_deliveries_total URL metadata deliveries sent to peers, by delivery outcome.
# TYPE yacynode_rwidistribution_url_metadata_deliveries_total counter
yacynode_rwidistribution_url_metadata_deliveries_total{result="accepted"} 2
yacynode_rwidistribution_url_metadata_deliveries_total{result="deferred"} 1
# HELP yacynode_rwidistribution_urls_delivered_total URLs delivered to peers as metadata, by delivery outcome.
# TYPE yacynode_rwidistribution_urls_delivered_total counter
yacynode_rwidistribution_urls_delivered_total{result="accepted"} 5
yacynode_rwidistribution_urls_delivered_total{result="deferred"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_url_metadata_deliveries_total",
		"yacynode_rwidistribution_urls_delivered_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsStaleReplicasDropped(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveStaleReplicasDropped(2)

	expected := `
# HELP yacynode_rwidistribution_stale_replicas_dropped_total Replicas dropped for peers no longer responsible.
# TYPE yacynode_rwidistribution_stale_replicas_dropped_total counter
yacynode_rwidistribution_stale_replicas_dropped_total 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_stale_replicas_dropped_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsSkippedBatchesByReason(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveBatchSkipped("too_few_reachable_peers")
	observer.ObserveBatchSkipped("too_few_reachable_peers")

	expected := `
# HELP yacynode_rwidistribution_batches_skipped_total Distribution batches the node did not start, by reason.
# TYPE yacynode_rwidistribution_batches_skipped_total counter
yacynode_rwidistribution_batches_skipped_total{reason="too_few_reachable_peers"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_batches_skipped_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsCompletedBatches(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveBatchCompleted()
	observer.ObserveBatchCompleted()

	expected := `
# HELP yacynode_rwidistribution_batches_completed_total Distribution batches that recorded their outcome or found no posting to send.
# TYPE yacynode_rwidistribution_batches_completed_total counter
yacynode_rwidistribution_batches_completed_total 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_batches_completed_total",
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
# HELP yacynode_rwidistribution_batches_aborted_total Distribution batches aborted before their outcome was recorded, by reason.
# TYPE yacynode_rwidistribution_batches_aborted_total counter
yacynode_rwidistribution_batches_aborted_total{reason="due_postings_unread"} 1
yacynode_rwidistribution_batches_aborted_total{reason="not_written"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_batches_aborted_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionTracksUndeliveredPostings(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveUndeliveredPostings(7)
	observer.ObserveUndeliveredPostings(4)

	expected := `
# HELP yacynode_rwidistribution_undelivered_postings Postings not yet delivered to all their responsible peers.
# TYPE yacynode_rwidistribution_undelivered_postings gauge
yacynode_rwidistribution_undelivered_postings 4
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_undelivered_postings",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionTracksTheLongestRefreshLateness(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveLongestRefreshLateness(90 * time.Second)

	expected := `
# HELP yacynode_rwidistribution_longest_refresh_lateness_seconds How long the most overdue word partition refresh has waited past its refresh time.
# TYPE yacynode_rwidistribution_longest_refresh_lateness_seconds gauge
yacynode_rwidistribution_longest_refresh_lateness_seconds 90
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_longest_refresh_lateness_seconds",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsPostingsHandedOff(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePostingsHandedOff(3)
	observer.ObservePostingsHandedOff(0)
	observer.ObservePostingsHandedOff(2)

	expected := `
# HELP yacynode_rwidistribution_postings_handed_off_total Postings deleted after peers closer to their DHT position accepted them.
# TYPE yacynode_rwidistribution_postings_handed_off_total counter
yacynode_rwidistribution_postings_handed_off_total 5
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_postings_handed_off_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsExcludedPeers(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePeerExcluded()
	observer.ObservePeerExcluded()

	expected := `
# HELP yacynode_rwidistribution_peers_excluded_total Peers excluded from distribution after failed offers in a row.
# TYPE yacynode_rwidistribution_peers_excluded_total counter
yacynode_rwidistribution_peers_excluded_total 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_peers_excluded_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionCountsSkippedWordPartitionsByOfferAndReason(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveWordPartitionSkipped("undelivered", "recipient_in_cooldown")
	observer.ObserveWordPartitionSkipped("undelivered", "recipient_in_cooldown")
	observer.ObserveWordPartitionSkipped("refresh", "no_recipient")

	expected := `
# HELP yacynode_rwidistribution_word_partitions_skipped_total Word partitions the node read but did not send, by offer and reason. They wait for the next pass over their ring sector.
# TYPE yacynode_rwidistribution_word_partitions_skipped_total counter
yacynode_rwidistribution_word_partitions_skipped_total{offer="refresh",reason="no_recipient"} 1
yacynode_rwidistribution_word_partitions_skipped_total{offer="undelivered",reason="recipient_in_cooldown"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_word_partitions_skipped_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
