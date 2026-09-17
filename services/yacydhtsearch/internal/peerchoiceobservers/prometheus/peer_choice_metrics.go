// Package prometheus reports how near the peers a query asks sit to the places
// on the DHT ring where the postings of its words belong.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type PeerChoiceMetrics struct {
	ringFractionFromWordToPeer prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *PeerChoiceMetrics {
	metrics := &PeerChoiceMetrics{
		ringFractionFromWordToPeer: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_selection_ring_fraction",
				Help: "Fraction of the DHT ring between the place where the postings " +
					"of a query word belong and the peer picked to answer for them.",
				Buckets: prometheusclient.ExponentialBucketsRange(1e-6, 1, 13),
			},
		),
	}
	registry.MustRegister(metrics.ringFractionFromWordToPeer)

	return metrics
}

func (m *PeerChoiceMetrics) PeersTakenFromTheRing(_ context.Context, ringFractions []float64) {
	for _, fraction := range ringFractions {
		m.ringFractionFromWordToPeer.Observe(fraction)
	}
}
