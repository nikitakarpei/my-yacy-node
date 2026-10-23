// Package prometheus counts the searches the text corpus answers and fails, and how long
// the search index takes to answer them.
package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

const labelSearchOutcome = "outcome"

const (
	outcomeAnswered = "answered"
	outcomeFailed   = "failed"
)

var searchOutcomes = []string{
	outcomeAnswered,
	outcomeFailed,
}

type TextSearchMetrics struct {
	searches           *prometheus.CounterVec
	searchDurationSecs prometheus.Histogram
}

func New(registry prometheus.Registerer) *TextSearchMetrics {
	metrics := &TextSearchMetrics{
		searches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpustext_searches_total",
			Help: "Text searches, by outcome.",
		}, []string{labelSearchOutcome}),
		searchDurationSecs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "corpustext_search_duration_seconds",
			Help:    "Search-index search duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	for _, outcome := range searchOutcomes {
		metrics.searches.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.searches, metrics.searchDurationSecs)
	return metrics
}

func (m *TextSearchMetrics) SearchAnswered(
	_ context.Context,
	_ textsearch.Search,
	elapsed time.Duration,
	_ int,
) {
	m.countSearch(outcomeAnswered, elapsed)
}

func (m *TextSearchMetrics) SearchFailed(
	_ context.Context,
	_ textsearch.Search,
	elapsed time.Duration,
	_ error,
) {
	m.countSearch(outcomeFailed, elapsed)
}

func (m *TextSearchMetrics) countSearch(outcome string, elapsed time.Duration) {
	m.searches.WithLabelValues(outcome).Inc()
	m.searchDurationSecs.Observe(elapsed.Seconds())
}
