package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelResult  = "result"
	resultFound  = "found"
	resultMissed = "missed"
)

type RobotsRulesMetrics struct {
	robotsRulesLookups *prometheusclient.CounterVec
	robotsRulesNotRead prometheusclient.Counter
	robotsRulesNotKept prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *RobotsRulesMetrics {
	metrics := &RobotsRulesMetrics{
		robotsRulesLookups: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacycrawler_robots_rules_lookups_total",
			Help: "Lookups of the kept robots rules of a site, by whether the rules were kept.",
		}, []string{labelResult}),
		robotsRulesNotRead: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_robots_rules_not_read_total",
			Help: "Kept robots rules the crawler could not read, so it fetched robots.txt again.",
		}),
		robotsRulesNotKept: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "yacycrawler_robots_rules_not_kept_total",
			Help: "Robots rules the crawler could not keep for the other pages of their site.",
		}),
	}
	for _, result := range []string{resultFound, resultMissed} {
		metrics.robotsRulesLookups.WithLabelValues(result)
	}
	registry.MustRegister(
		metrics.robotsRulesLookups, metrics.robotsRulesNotRead, metrics.robotsRulesNotKept,
	)
	return metrics
}

func (metrics *RobotsRulesMetrics) RobotsRulesFound(context.Context, canonicalurl.CanonicalURL) {
	metrics.robotsRulesLookups.WithLabelValues(resultFound).Inc()
}

func (metrics *RobotsRulesMetrics) RobotsRulesMissed(context.Context, canonicalurl.CanonicalURL) {
	metrics.robotsRulesLookups.WithLabelValues(resultMissed).Inc()
}

func (metrics *RobotsRulesMetrics) RobotsRulesNotRead(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.robotsRulesNotRead.Inc()
}

func (metrics *RobotsRulesMetrics) RobotsRulesNotKept(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	metrics.robotsRulesNotKept.Inc()
}
