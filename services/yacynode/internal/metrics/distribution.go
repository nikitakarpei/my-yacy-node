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
	postingsGone          prometheus.Counter
	scheduledPostings     prometheus.Gauge
	longestOfferLateness  prometheus.Gauge
	staleReplicasDropped  prometheus.Counter
	postingsHandedOff     prometheus.Counter
	cyclesSkipped         *prometheus.CounterVec
	batchesAborted        *prometheus.CounterVec
}

func NewDistributionMetrics(registry prometheus.Registerer) *DistributionMetrics {
	postingOffers := counterFor(
		"rwidistribution_posting_offers_sent_total",
		"Posting offers sent to peers, by offer outcome.",
		labelOutcome,
	)
	postingsOffered := counterFor(
		"rwidistribution_postings_offered_total",
		"RWI postings offered to peers, by offer outcome.",
		labelOutcome,
	)
	urlMetadataDeliveries := counterFor(
		"rwidistribution_url_metadata_deliveries_total",
		"URL metadata deliveries sent to peers, by delivery outcome.",
		labelOutcome,
	)
	urlsDelivered := counterFor(
		"rwidistribution_urls_delivered_total",
		"URLs delivered to peers as metadata, by delivery outcome.",
		labelOutcome,
	)
	postingsGone := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_postings_gone_total",
		Help: "Due postings evicted between the schedule read and the posting read.",
	})
	scheduledPostings := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rwidistribution_scheduled_postings",
		Help: "Postings holding a due entry on the offer schedule.",
	})
	longestOfferLateness := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rwidistribution_longest_offer_lateness_seconds",
		Help: "Time the most overdue posting offer is past its scheduled time.",
	})
	staleReplicasDropped := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_stale_replicas_dropped_total",
		Help: "Replicas dropped for peers no longer responsible.",
	})
	postingsHandedOff := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_postings_handed_off_total",
		Help: "Postings deleted after peers closer to their DHT position accepted them.",
	})
	cyclesSkipped := counterFor(
		"rwidistribution_cycles_skipped_total",
		"Distribution cycles that ran no batch, by reason.",
		labelReason,
	)
	batchesAborted := counterFor(
		"rwidistribution_batches_aborted_total",
		"Distribution batches aborted before their postings were rescheduled, by reason.",
		labelReason,
	)
	registry.MustRegister(
		postingOffers, postingsOffered, urlMetadataDeliveries, urlsDelivered,
		postingsGone, scheduledPostings, longestOfferLateness, staleReplicasDropped,
		postingsHandedOff, cyclesSkipped, batchesAborted,
	)

	return &DistributionMetrics{
		postingOffers:         postingOffers,
		postingsOffered:       postingsOffered,
		urlMetadataDeliveries: urlMetadataDeliveries,
		urlsDelivered:         urlsDelivered,
		postingsGone:          postingsGone,
		scheduledPostings:     scheduledPostings,
		longestOfferLateness:  longestOfferLateness,
		staleReplicasDropped:  staleReplicasDropped,
		postingsHandedOff:     postingsHandedOff,
		cyclesSkipped:         cyclesSkipped,
		batchesAborted:        batchesAborted,
	}
}

func counterFor(name string, help string, label string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: help}, []string{label})
}

func (d *DistributionMetrics) ObservePostingOffer(outcome string, postings int) {
	d.postingOffers.WithLabelValues(outcome).Inc()
	d.postingsOffered.WithLabelValues(outcome).Add(float64(postings))
}

func (d *DistributionMetrics) ObserveURLMetadataDelivery(outcome string, urls int) {
	d.urlMetadataDeliveries.WithLabelValues(outcome).Inc()
	d.urlsDelivered.WithLabelValues(outcome).Add(float64(urls))
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

func (d *DistributionMetrics) ObserveBatchAborted(reason string) {
	d.batchesAborted.WithLabelValues(reason).Inc()
}
