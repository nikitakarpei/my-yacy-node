package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelOutcome    = "result"
	labelSkipReason = "reason"
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
}

func NewDistributionMetrics(registry prometheus.Registerer) *DistributionMetrics {
	postingOffers := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rwidistribution_posting_offers_sent_total",
			Help: "Posting offers sent to peers, by offer outcome.",
		},
		[]string{labelOutcome},
	)
	postingsOffered := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rwidistribution_postings_offered_total",
			Help: "RWI postings offered to peers, by offer outcome.",
		},
		[]string{labelOutcome},
	)
	urlMetadataDeliveries := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rwidistribution_url_metadata_deliveries_total",
			Help: "URL metadata deliveries sent to peers, by delivery outcome.",
		},
		[]string{labelOutcome},
	)
	urlsDelivered := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rwidistribution_urls_delivered_total",
			Help: "URLs delivered to peers as metadata, by delivery outcome.",
		},
		[]string{labelOutcome},
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
	cyclesSkipped := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rwidistribution_cycles_skipped_total",
			Help: "Distribution cycles that offered nothing, by reason.",
		},
		[]string{labelSkipReason},
	)
	registry.MustRegister(
		postingOffers, postingsOffered, urlMetadataDeliveries, urlsDelivered,
		postingsGone, scheduledPostings, longestOfferLateness, staleReplicasDropped,
		postingsHandedOff, cyclesSkipped,
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
	}
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
