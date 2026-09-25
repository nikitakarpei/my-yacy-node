// Package prometheus publishes the URL metadata ask ceilings handed out for the
// asks to the peers.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type AskCeilingMetrics struct {
	handedOutCeilings prometheusclient.Histogram
}

func New(registry prometheusclient.Registerer) *AskCeilingMetrics {
	handedOutCeilings := prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
		Name:    "yacydhtsearch_url_metadata_ask_ceiling_documents",
		Help:    "Most documents one URL metadata ask to a peer may name, per ask.",
		Buckets: []float64{25, 50, 100, 200, 400, 700, 1000},
	})
	registry.MustRegister(handedOutCeilings)

	return &AskCeilingMetrics{handedOutCeilings: handedOutCeilings}
}

func (*AskCeilingMetrics) AskCeilingSet(context.Context, string, float64, int) {}

func (m *AskCeilingMetrics) AskCeilingHandedOut(_ context.Context, _ string, ceiling int) {
	m.handedOutCeilings.Observe(float64(ceiling))
}
