// Package prometheus reports how near the picked peers sit to the postings of a
// query's terms on the DHT ring.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type DHTDistanceMetrics struct {
	ringFractionFromTermToPeer prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *DHTDistanceMetrics {
	metrics := &DHTDistanceMetrics{
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

func (m *DHTDistanceMetrics) PeersSelected(_ context.Context, ringFractions []float64) {
	for _, fraction := range ringFractions {
		m.ringFractionFromTermToPeer.Observe(fraction)
	}
}
