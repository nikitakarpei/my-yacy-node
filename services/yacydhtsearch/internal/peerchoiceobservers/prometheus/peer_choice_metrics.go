// Package prometheus reports how near the peers a query asks sit to the
// postings of its terms on the DHT ring.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type PeerChoiceMetrics struct {
	ringFractionFromTermToPeer prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *PeerChoiceMetrics {
	metrics := &PeerChoiceMetrics{
		ringFractionFromTermToPeer: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_selection_ring_fraction",
				Help: "Fraction of the DHT ring between the postings of a query term " +
					"and the peer picked to answer for them.",
				Buckets: prometheusclient.ExponentialBucketsRange(1e-6, 1, 13),
			},
		),
	}
	registry.MustRegister(metrics.ringFractionFromTermToPeer)

	return metrics
}

func (m *PeerChoiceMetrics) PeersSelected(_ context.Context, ringFractions []float64) {
	for _, fraction := range ringFractions {
		m.ringFractionFromTermToPeer.Observe(fraction)
	}
}
