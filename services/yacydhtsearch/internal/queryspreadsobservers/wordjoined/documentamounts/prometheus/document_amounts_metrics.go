// Package prometheus reports where the document amounts of the query words came
// from, as metrics.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
)

const (
	labelSource = "source"
	sourceCache = "cache"
	sourceNone  = "none"
)

type DocumentAmountsMetrics struct {
	amountsFromCache    prometheusclient.Counter
	amountsFromNoSource prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *DocumentAmountsMetrics {
	sources := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_word_joined_spread_document_amount_sources_total",
		Help: "Document amounts of the query words, by where they came from; " +
			"none when a query word had no remembered amount.",
	}, []string{labelSource})
	metrics := &DocumentAmountsMetrics{
		amountsFromCache:    sources.WithLabelValues(sourceCache),
		amountsFromNoSource: sources.WithLabelValues(sourceNone),
	}
	registry.MustRegister(sources)

	return metrics
}

func (m *DocumentAmountsMetrics) AmountsReadFromCache(
	_ context.Context,
	performed documentamounts.PerformedFromCache,
) {
	if !performed.AllQueryWordsCached {
		m.amountsFromNoSource.Inc()

		return
	}
	m.amountsFromCache.Inc()
}
