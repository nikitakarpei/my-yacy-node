package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const labelOutcome = "result"

type DistributionMetrics struct {
	offerRequests       *prometheus.CounterVec
	postingsOffered     *prometheus.CounterVec
	postingsDue         prometheus.Counter
	postingsGone        prometheus.Counter
	oldestDuePostingAge prometheus.Gauge
	ledgerPrunes        prometheus.Counter
	cyclesSkipped       prometheus.Counter
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
		offerRequests, postingsOffered, postingsDue, postingsGone,
		oldestDuePostingAge, ledgerPrunes, cyclesSkipped,
	)

	return &DistributionMetrics{
		offerRequests:       offerRequests,
		postingsOffered:     postingsOffered,
		postingsDue:         postingsDue,
		postingsGone:        postingsGone,
		oldestDuePostingAge: oldestDuePostingAge,
		ledgerPrunes:        ledgerPrunes,
		cyclesSkipped:       cyclesSkipped,
	}
}

func (d *DistributionMetrics) ObservePostingOffer(outcome string, postings int) {
	d.offerRequests.WithLabelValues(outcome).Inc()
	d.postingsOffered.WithLabelValues(outcome).Add(float64(postings))
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
