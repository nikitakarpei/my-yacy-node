// Package prometheus counts the page calls the service answers, split by what became of the
// fetch each one asked for, so an operator can tell a stack that reads pages from one that
// gives them up or never finishes them.
package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

const fetchOutcomeLabel = "fetch_outcome"

type PageReadMetrics struct {
	pagesAnswered          *prometheus.CounterVec
	markdownRecallFailures prometheus.Counter
	fetchOutcomesNotHeard  prometheus.Counter
}

func New(registry prometheus.Registerer) *PageReadMetrics {
	metrics := &PageReadMetrics{
		pagesAnswered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "webresearchmcp_pages_answered_total",
			Help: "Page calls answered, by what became of the fetch each one asked for.",
		}, []string{fetchOutcomeLabel}),
		markdownRecallFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_markdown_recall_failures_total",
			Help: "Reads from the markdown corpus that failed and left the caller with an error.",
		}),
		fetchOutcomesNotHeard: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_fetch_outcomes_not_heard_total",
			Help: "Page calls that never heard what became of the fetch they asked for.",
		}),
	}
	registry.MustRegister(
		metrics.pagesAnswered,
		metrics.markdownRecallFailures,
		metrics.fetchOutcomesNotHeard,
	)
	return metrics
}

func (m *PageReadMetrics) PageAnswered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	fetchOutcome pageread.FetchOutcome,
) {
	m.pagesAnswered.WithLabelValues(string(fetchOutcome)).Inc()
}

func (m *PageReadMetrics) MarkdownRecallFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.markdownRecallFailures.Inc()
}

func (m *PageReadMetrics) FetchOutcomeNotHeard(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	m.fetchOutcomesNotHeard.Inc()
}
