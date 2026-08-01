package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const labelOutcome = "result"

type DistributionMetrics struct {
	postingOffers         *prometheus.CounterVec
	postingsOffered       *prometheus.CounterVec
	urlMetadataDeliveries *prometheus.CounterVec
	urlsDelivered         *prometheus.CounterVec
	postingsDue           prometheus.Counter
	postingsGone          prometheus.Counter
	oldestDuePostingAge   prometheus.Gauge
	ledgerPrunes          prometheus.Counter
	cyclesSkipped         prometheus.Counter
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
	postingsDue := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_postings_due_total",
		Help: "Postings the schedule reported as due for an offer, summed across cycles.",
	})
	postingsGone := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_postings_gone_total",
		Help: "Due postings that no longer exist in the posting index when read for offering.",
	})
	oldestDuePostingAge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rwidistribution_oldest_due_posting_age_seconds",
		Help: "Age of the oldest posting still awaiting an offer.",
	})
	ledgerPrunes := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_ledger_prunes_total",
		Help: "Replica ledger entries dropped for peers no longer responsible.",
	})
	cyclesSkipped := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_cycles_skipped_total",
		Help: "Distribution cycles skipped because too few peers were reachable.",
	})
	registry.MustRegister(
		postingOffers, postingsOffered, urlMetadataDeliveries, urlsDelivered,
		postingsDue, postingsGone, oldestDuePostingAge, ledgerPrunes, cyclesSkipped,
	)

	return &DistributionMetrics{
		postingOffers:         postingOffers,
		postingsOffered:       postingsOffered,
		urlMetadataDeliveries: urlMetadataDeliveries,
		urlsDelivered:         urlsDelivered,
		postingsDue:           postingsDue,
		postingsGone:          postingsGone,
		oldestDuePostingAge:   oldestDuePostingAge,
		ledgerPrunes:          ledgerPrunes,
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

func (d *DistributionMetrics) ObservePostingsDue(due int) {
	d.postingsDue.Add(float64(due))
}

func (d *DistributionMetrics) ObservePostingsGone(gone int) {
	d.postingsGone.Add(float64(gone))
}

func (d *DistributionMetrics) ObserveOldestDuePostingAge(age time.Duration) {
	d.oldestDuePostingAge.Set(age.Seconds())
}

func (d *DistributionMetrics) ObserveLedgerPrune(dropped int) {
	d.ledgerPrunes.Add(float64(dropped))
}

func (d *DistributionMetrics) ObserveCycleSkipped() {
	d.cyclesSkipped.Inc()
}
