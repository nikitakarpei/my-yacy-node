// Package prometheus reports how often the query word document amounts failed.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	labelAction  = "action"
	actionLookup = "lookup"
	actionStore  = "store"
)

type QueryWordDocumentAmountsMetrics struct {
	lookupFailures prometheusclient.Counter
	storeFailures  prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *QueryWordDocumentAmountsMetrics {
	failures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_query_word_document_amounts_failures_total",
		Help: "Failures against the remembered query word document amounts, by action.",
	}, []string{labelAction})
	registry.MustRegister(failures)

	return &QueryWordDocumentAmountsMetrics{
		lookupFailures: failures.WithLabelValues(actionLookup),
		storeFailures:  failures.WithLabelValues(actionStore),
	}
}

func (m *QueryWordDocumentAmountsMetrics) DocumentAmountLookupFailed(
	context.Context,
	yacymodel.Hash,
	error,
) {
	m.lookupFailures.Inc()
}

func (m *QueryWordDocumentAmountsMetrics) DocumentAmountStoreFailed(
	context.Context,
	yacymodel.Hash,
	error,
) {
	m.storeFailures.Inc()
}
