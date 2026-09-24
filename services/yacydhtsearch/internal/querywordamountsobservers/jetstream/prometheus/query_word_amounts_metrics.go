// Package prometheus reports how often the query word amounts failed.
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

type QueryWordAmountsMetrics struct {
	lookupFailures prometheusclient.Counter
	storeFailures  prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *QueryWordAmountsMetrics {
	failures := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_query_word_amounts_failures_total",
		Help: "Failures against the remembered query word amounts, by action.",
	}, []string{labelAction})
	registry.MustRegister(failures)

	return &QueryWordAmountsMetrics{
		lookupFailures: failures.WithLabelValues(actionLookup),
		storeFailures:  failures.WithLabelValues(actionStore),
	}
}

func (m *QueryWordAmountsMetrics) AmountLookupFailed(context.Context, yacymodel.Hash, error) {
	m.lookupFailures.Inc()
}

func (m *QueryWordAmountsMetrics) AmountStoreFailed(context.Context, yacymodel.Hash, error) {
	m.storeFailures.Inc()
}
