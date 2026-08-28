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

type PageReadProgressMetrics struct {
	pagesAnswered               *prometheus.CounterVec
	markdownRecallFailures      prometheus.Counter
	scrapeOutcomeListenFailures prometheus.Counter
	scrapeRequestFailures       prometheus.Counter
	fetchOutcomesNotHeard       prometheus.Counter
}

func New(registry prometheus.Registerer) *PageReadProgressMetrics {
	metrics := &PageReadProgressMetrics{
		pagesAnswered: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "webresearchmcp_pages_answered_total",
			Help: "Page calls answered, by what became of the fetch each one asked for.",
		}, []string{fetchOutcomeLabel}),
		markdownRecallFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_markdown_recall_failures_total",
			Help: "Reads from the markdown corpus that failed and left the caller with an error.",
		}),
		scrapeOutcomeListenFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_scrape_outcome_listen_failures_total",
			Help: "Page calls that could not listen for the outcome of the fetch they wanted.",
		}),
		scrapeRequestFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_scrape_request_failures_total",
			Help: "Page calls whose scrape request the broker did not take.",
		}),
		fetchOutcomesNotHeard: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "webresearchmcp_fetch_outcomes_not_heard_total",
			Help: "Page calls that never heard what became of the fetch they asked for.",
		}),
	}
	registry.MustRegister(
		metrics.pagesAnswered,
		metrics.markdownRecallFailures,
		metrics.scrapeOutcomeListenFailures,
		metrics.scrapeRequestFailures,
		metrics.fetchOutcomesNotHeard,
	)
	return metrics
}

func (m *PageReadProgressMetrics) PageAnswered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	fetchOutcome pageread.FetchOutcome,
) {
	m.pagesAnswered.WithLabelValues(string(fetchOutcome)).Inc()
}

func (m *PageReadProgressMetrics) MarkdownRecallFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.markdownRecallFailures.Inc()
}

func (m *PageReadProgressMetrics) ScrapeOutcomeListenFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.scrapeOutcomeListenFailures.Inc()
}

func (m *PageReadProgressMetrics) ScrapeRequestFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.scrapeRequestFailures.Inc()
}

func (m *PageReadProgressMetrics) FetchOutcomeNotHeard(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
	_ error,
) {
	m.fetchOutcomesNotHeard.Inc()
}
