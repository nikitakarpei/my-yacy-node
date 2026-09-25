// Package prometheus reports how many found documents ranked lower because
// their host is not among the well-linked hosts.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type RelevanceMetrics struct {
	demotedDocuments prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RelevanceMetrics {
	metrics := &RelevanceMetrics{
		demotedDocuments: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_well_linked_host_demoted_documents_total",
			Help: "Found documents whose relevance fell because their host " +
				"is not among the well-linked hosts.",
		}),
	}
	registry.MustRegister(metrics.demotedDocuments)

	return metrics
}

func (m *RelevanceMetrics) DocumentsDemoted(_ context.Context, amountOfDemotedDocuments int) {
	m.demotedDocuments.Add(float64(amountOfDemotedDocuments))
}
