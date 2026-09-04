// Package prometheus counts what the service meets while it waits for the end of one scrape,
// so an operator can tell a wait that never opened from one that a message it cannot read
// keeps going.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome = "outcome"

	outcomeSubscriptionFailed         = "subscription_failed"
	outcomeListenerConfirmationFailed = "listener_confirmation_failed"
	outcomeMessageMalformed           = "message_malformed"
)

var scrapeOutcomeListeningOutcomes = []string{
	outcomeSubscriptionFailed,
	outcomeListenerConfirmationFailed,
	outcomeMessageMalformed,
}

type ScrapeOutcomeMetrics struct{ listeningFailures *prometheusclient.CounterVec }

func New(registry prometheusclient.Registerer) *ScrapeOutcomeMetrics {
	metrics := &ScrapeOutcomeMetrics{listeningFailures: prometheusclient.NewCounterVec(
		prometheusclient.CounterOpts{
			Name: "webresearchmcp_scrape_outcome_listening_failures_total",
			Help: "Failures met while listening for the end of one scrape, by outcome.",
		}, []string{labelOutcome},
	)}
	for _, outcome := range scrapeOutcomeListeningOutcomes {
		metrics.listeningFailures.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.listeningFailures)
	return metrics
}

func (metrics *ScrapeOutcomeMetrics) ScrapeOutcomeSubscriptionFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.count(outcomeSubscriptionFailed)
}

func (metrics *ScrapeOutcomeMetrics) ScrapeOutcomeListenerConfirmationFailed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.count(outcomeListenerConfirmationFailed)
}

func (metrics *ScrapeOutcomeMetrics) ScrapeOutcomeMessageMalformed(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.count(outcomeMessageMalformed)
}

func (metrics *ScrapeOutcomeMetrics) count(outcome string) {
	metrics.listeningFailures.WithLabelValues(outcome).Inc()
}
