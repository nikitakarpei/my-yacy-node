package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const labelOutcome = "result"

type DistributionMetrics struct {
	offerRequests   *prometheus.CounterVec
	postingsOffered *prometheus.CounterVec
	scheduleDrained prometheus.Counter
	ledgerPrunes    prometheus.Counter
	cyclesSkipped   prometheus.Counter
}

func NewDistributionMetrics(registry prometheus.Registerer) *DistributionMetrics {
	offerRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rwidistribution_offer_requests_total",
			Help: "Offer requests sent to peers, by offer outcome.",
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
	scheduleDrained := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_schedule_drained_total",
		Help: "Due postings drained from the distribution schedule for offering.",
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
		offerRequests, postingsOffered, scheduleDrained, ledgerPrunes, cyclesSkipped,
	)

	return &DistributionMetrics{
		offerRequests:   offerRequests,
		postingsOffered: postingsOffered,
		scheduleDrained: scheduleDrained,
		ledgerPrunes:    ledgerPrunes,
		cyclesSkipped:   cyclesSkipped,
	}
}

func (d *DistributionMetrics) ObservePostingOffer(outcome string, postings int) {
	d.offerRequests.WithLabelValues(outcome).Inc()
	d.postingsOffered.WithLabelValues(outcome).Add(float64(postings))
}

func (d *DistributionMetrics) ObserveScheduleDrain(drained int) {
	d.scheduleDrained.Add(float64(drained))
}

func (d *DistributionMetrics) ObserveLedgerPrune(dropped int) {
	d.ledgerPrunes.Add(float64(dropped))
}

func (d *DistributionMetrics) ObserveCycleSkipped(int) {
	d.cyclesSkipped.Inc()
}
