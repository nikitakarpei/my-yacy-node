package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome       = "outcome"
	outcomeAnswered    = "answered"
	outcomeUnreachable = "unreachable"
	outcomeCanceled    = "canceled"
)

type RobotsTxtFetchMetrics struct {
	robotsTxtFetches *prometheusclient.CounterVec
}

func New(registry prometheusclient.Registerer) *RobotsTxtFetchMetrics {
	metrics := &RobotsTxtFetchMetrics{
		robotsTxtFetches: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_robots_txt_fetches_total",
			Help: "Fetches of robots.txt, by whether the site answered.",
		}, []string{labelOutcome}),
	}
	for _, outcome := range []string{outcomeAnswered, outcomeUnreachable, outcomeCanceled} {
		metrics.robotsTxtFetches.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.robotsTxtFetches)
	return metrics
}

func (metrics *RobotsTxtFetchMetrics) RobotsTxtAnswered(
	context.Context, canonicalurl.CanonicalURL,
) {
	metrics.robotsTxtFetches.WithLabelValues(outcomeAnswered).Inc()
}

func (metrics *RobotsTxtFetchMetrics) RobotsTxtUnreachable(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.robotsTxtFetches.WithLabelValues(outcomeUnreachable).Inc()
}

func (metrics *RobotsTxtFetchMetrics) RobotsTxtFetchCanceled(
	context.Context, canonicalurl.CanonicalURL,
) {
	metrics.robotsTxtFetches.WithLabelValues(outcomeCanceled).Inc()
}
