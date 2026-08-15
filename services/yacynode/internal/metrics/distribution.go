package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome = "result"
	labelReason  = "reason"
)

type DistributionMetrics struct {
	postingOffers         *prometheus.CounterVec
	postingsOffered       *prometheus.CounterVec
	urlMetadataDeliveries *prometheus.CounterVec
	urlsDelivered         *prometheus.CounterVec
	urlsUnknownToUs       prometheus.Counter
	postingsGone          prometheus.Counter
	scheduledPostings     prometheus.Gauge
	longestOfferLateness  prometheus.Gauge
	staleReplicasDropped  prometheus.Counter
	postingsHandedOff     prometheus.Counter
	cyclesSkipped         *prometheus.CounterVec
	cyclesCompleted       prometheus.Counter
	batchesAborted        *prometheus.CounterVec
}

func NewDistributionMetrics(registry prometheus.Registerer) *DistributionMetrics {
	metrics := distributionMetrics()
	registry.MustRegister(metrics.collectors()...)

	return metrics
}

func distributionMetrics() *DistributionMetrics {
	return &DistributionMetrics{
		postingOffers: counterPerLabelFor(
			"rwidistribution_posting_offers_sent_total",
			"Posting offers sent to peers, by offer outcome.",
			labelOutcome,
		),
		postingsOffered: counterPerLabelFor(
			"rwidistribution_postings_offered_total",
			"RWI postings offered to peers, by offer outcome.",
			labelOutcome,
		),
		urlMetadataDeliveries: counterPerLabelFor(
			"rwidistribution_url_metadata_deliveries_total",
			"URL metadata deliveries sent to peers, by delivery outcome.",
			labelOutcome,
		),
		urlsDelivered: counterPerLabelFor(
			"rwidistribution_urls_delivered_total",
			"URLs delivered to peers as metadata, by delivery outcome.",
			labelOutcome,
		),
		urlsUnknownToUs: counterFor(
			"rwidistribution_urls_unknown_to_us_total",
			"URLs a peer asked for whose metadata this node does not hold, so no delivery carried them.",
		),
		postingsGone: counterFor(
			"rwidistribution_postings_gone_total",
			"Due postings evicted between the schedule read and the posting read.",
		),
		scheduledPostings: gaugeFor(
			"rwidistribution_scheduled_postings",
			"Postings holding a due entry on the offer schedule.",
		),
		longestOfferLateness: gaugeFor(
			"rwidistribution_longest_offer_lateness_seconds",
			"Time the most overdue posting offer is past its scheduled time.",
		),
		staleReplicasDropped: counterFor(
			"rwidistribution_stale_replicas_dropped_total",
			"Replicas dropped for peers no longer responsible.",
		),
		postingsHandedOff: counterFor(
			"rwidistribution_postings_handed_off_total",
			"Postings deleted after peers closer to their DHT position accepted them.",
		),
		cyclesSkipped: counterPerLabelFor(
			"rwidistribution_cycles_skipped_total",
			"Distribution cycles that ran no batch, by reason.",
			labelReason,
		),
		cyclesCompleted: counterFor(
			"rwidistribution_cycles_completed_total",
			"Distribution cycles that ran to the end of their due postings.",
		),
		batchesAborted: counterPerLabelFor(
			"rwidistribution_batches_aborted_total",
			"Distribution batches aborted before their postings were rescheduled, by reason.",
			labelReason,
		),
	}
}

func (d *DistributionMetrics) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		d.postingOffers,
		d.postingsOffered,
		d.urlMetadataDeliveries,
		d.urlsDelivered,
		d.urlsUnknownToUs,
		d.postingsGone,
		d.scheduledPostings,
		d.longestOfferLateness,
		d.staleReplicasDropped,
		d.postingsHandedOff,
		d.cyclesSkipped,
		d.cyclesCompleted,
		d.batchesAborted,
	}
}

func counterPerLabelFor(name string, help string, label string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, []string{label})
}

func counterFor(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{Name: name, Help: help})
}

func gaugeFor(name string, help string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{Name: name, Help: help})
}

func (d *DistributionMetrics) ObservePostingOffer(outcome string, postings int) {
	d.postingOffers.WithLabelValues(outcome).Inc()
	d.postingsOffered.WithLabelValues(outcome).Add(float64(postings))
}

func (d *DistributionMetrics) ObserveURLMetadataDelivery(outcome string, urls int) {
	d.urlMetadataDeliveries.WithLabelValues(outcome).Inc()
	d.urlsDelivered.WithLabelValues(outcome).Add(float64(urls))
}

func (d *DistributionMetrics) ObserveURLsUnknownToUs(urls int) {
	d.urlsUnknownToUs.Add(float64(urls))
}

func (d *DistributionMetrics) ObservePostingsGone(gone int) {
	d.postingsGone.Add(float64(gone))
}

func (d *DistributionMetrics) ObserveScheduledPostings(postings int) {
	d.scheduledPostings.Set(float64(postings))
}

func (d *DistributionMetrics) ObserveLongestOfferLateness(lateness time.Duration) {
	d.longestOfferLateness.Set(lateness.Seconds())
}

func (d *DistributionMetrics) ObserveStaleReplicasDropped(dropped int) {
	d.staleReplicasDropped.Add(float64(dropped))
}

func (d *DistributionMetrics) ObservePostingsHandedOff(handedOff int) {
	d.postingsHandedOff.Add(float64(handedOff))
}

func (d *DistributionMetrics) ObserveCycleSkipped(reason string) {
	d.cyclesSkipped.WithLabelValues(reason).Inc()
}

func (d *DistributionMetrics) ObserveCycleCompleted() {
	d.cyclesCompleted.Inc()
}

func (d *DistributionMetrics) ObserveBatchAborted(reason string) {
	d.batchesAborted.WithLabelValues(reason).Inc()
}
