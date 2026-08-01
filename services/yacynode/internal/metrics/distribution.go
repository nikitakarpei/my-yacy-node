package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const labelOutcome = "result"

type DistributionMetrics struct {
	offerRequests      *prometheus.CounterVec
	postingsOffered    *prometheus.CounterVec
	postingsConsidered prometheus.Counter
	ledgerPrunes       prometheus.Counter
	cyclesSkipped      prometheus.Counter
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
	postingsConsidered := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rwidistribution_postings_considered_total",
		Help: "Due postings read from the distribution schedule for offering.",
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
		offerRequests, postingsOffered, postingsConsidered, ledgerPrunes, cyclesSkipped,
	)

	return &DistributionMetrics{
		offerRequests:      offerRequests,
		postingsOffered:    postingsOffered,
		postingsConsidered: postingsConsidered,
		ledgerPrunes:       ledgerPrunes,
		cyclesSkipped:      cyclesSkipped,
	}
}

func (d *DistributionMetrics) ObservePostingOffer(outcome string, postings int) {
	d.offerRequests.WithLabelValues(outcome).Inc()
	d.postingsOffered.WithLabelValues(outcome).Add(float64(postings))
}

func (d *DistributionMetrics) ObservePostingsConsidered(considered int) {
	d.postingsConsidered.Add(float64(considered))
}

func (d *DistributionMetrics) ObserveLedgerPrune(dropped int) {
	d.ledgerPrunes.Add(float64(dropped))
}

func (d *DistributionMetrics) ObserveCycleSkipped() {
	d.cyclesSkipped.Inc()
}
