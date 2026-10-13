package metrics_test

import (
	"maps"
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
yacynode_rwidistribution_posting_offers_sent_total{outcome="busy"} 1
yacynode_rwidistribution_posting_offers_sent_total{outcome="ok"} 2
# HELP yacynode_rwidistribution_postings_offered_total RWI postings offered to peers, by offer outcome.
# TYPE yacynode_rwidistribution_postings_offered_total counter
yacynode_rwidistribution_postings_offered_total{outcome="busy"} 1
yacynode_rwidistribution_postings_offered_total{outcome="ok"} 5
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
yacynode_rwidistribution_url_metadata_deliveries_total{outcome="accepted"} 2
yacynode_rwidistribution_url_metadata_deliveries_total{outcome="deferred"} 1
# HELP yacynode_rwidistribution_urls_delivered_total URLs delivered to peers as metadata, by delivery outcome.
# TYPE yacynode_rwidistribution_urls_delivered_total counter
yacynode_rwidistribution_urls_delivered_total{outcome="accepted"} 5
yacynode_rwidistribution_urls_delivered_total{outcome="deferred"} 1
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

func TestDistributionCountsStaleHoldersDropped(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveStaleHoldersDropped(2)

	expected := `
# HELP yacynode_rwidistribution_stale_holders_dropped_total Holders dropped because they are no longer responsible.
# TYPE yacynode_rwidistribution_stale_holders_dropped_total counter
yacynode_rwidistribution_stale_holders_dropped_total 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_stale_holders_dropped_total",
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

func TestDistributionCountsPostingsDelivered(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePostingsDelivered(4)
	observer.ObservePostingsDelivered(0)
	observer.ObservePostingsDelivered(1)

	expected := `
# HELP yacynode_rwidistribution_postings_delivered_total Postings accepted by all peers of their DHT window and marked delivered.
# TYPE yacynode_rwidistribution_postings_delivered_total counter
yacynode_rwidistribution_postings_delivered_total 5
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_postings_delivered_total",
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

func TestDistributionReportsWaitingWordPartitionsByReason(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObserveWordPartitionsWaiting("recipient_paused", 5)
	observer.ObserveWordPartitionsWaiting("recipient_paused", 3)
	observer.ObserveWordPartitionsWaiting("no_recipient", 1)

	expected := `
# HELP yacynode_rwidistribution_word_partitions_waiting Word partitions not sent over the last full read of the ring, by reason.
# TYPE yacynode_rwidistribution_word_partitions_waiting gauge
yacynode_rwidistribution_word_partitions_waiting{reason="no_recipient"} 1
yacynode_rwidistribution_word_partitions_waiting{reason="recipient_paused"} 3
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_word_partitions_waiting",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionTimesPeerRequestsByKind(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePostingOfferAnswerTime(200 * time.Millisecond)
	observer.ObservePostingOfferAnswerTime(time.Second)
	observer.ObserveURLMetadataDeliveryAnswerTime(300 * time.Millisecond)

	samples := histogramSamplesOf(t, registry, "yacynode_rwidistribution_peer_request_seconds")
	want := map[string]uint64{"request=postings": 2, "request=url_metadata": 1}
	if !maps.Equal(samples, want) {
		t.Fatalf("samples = %v, want %v", samples, want)
	}
}

func histogramSamplesOf(
	t *testing.T,
	registry *prometheus.Registry,
	name string,
) map[string]uint64 {
	t.Helper()

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	samples := make(map[string]uint64)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := make([]string, 0, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				labels = append(labels, label.GetName()+"="+label.GetValue())
			}
			samples[strings.Join(labels, ",")] = metric.GetHistogram().GetSampleCount()
		}
	}

	return samples
}

func TestDistributionTimesPeerSends(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePeerSendTime(2 * time.Second)

	samples := histogramSamplesOf(t, registry, "yacynode_rwidistribution_peer_send_seconds")
	if want := map[string]uint64{"": 1}; !maps.Equal(samples, want) {
		t.Fatalf("samples = %v, want %v", samples, want)
	}
}

func TestDistributionCountsPostingsPerSend(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePostingsPerSend(3)
	observer.ObservePostingsPerSend(30000)

	expected := `
# HELP yacynode_rwidistribution_postings_per_send Postings in one send: all postings of one batch to one peer.
# TYPE yacynode_rwidistribution_postings_per_send histogram
yacynode_rwidistribution_postings_per_send_bucket{le="1"} 0
yacynode_rwidistribution_postings_per_send_bucket{le="10"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="50"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="100"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="250"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="500"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="1000"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="2500"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="5000"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="10000"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="25000"} 1
yacynode_rwidistribution_postings_per_send_bucket{le="50000"} 2
yacynode_rwidistribution_postings_per_send_bucket{le="100000"} 2
yacynode_rwidistribution_postings_per_send_bucket{le="+Inf"} 2
yacynode_rwidistribution_postings_per_send_sum 30003
yacynode_rwidistribution_postings_per_send_count 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_postings_per_send",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDistributionReportsPeersAtTheirSendLimit(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDistributionMetrics(registry)

	observer.ObservePeersAtSendLimit(3)
	observer.ObservePeersAtSendLimit(1)

	expected := `
# HELP yacynode_rwidistribution_peers_at_send_limit Peers with as many sends in flight as they may have; word partitions that need them wait.
# TYPE yacynode_rwidistribution_peers_at_send_limit gauge
yacynode_rwidistribution_peers_at_send_limit 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_peers_at_send_limit",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
