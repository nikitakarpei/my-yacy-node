// Package prometheus reports how many page visits the frontier holds each time
// metrics are scraped.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
)

type Frontier interface {
	AmountPending(ctx context.Context) (int64, bool)
}

type FrontierMetrics struct {
	frontier          Frontier
	pendingPageVisits *prometheusclient.Desc
}

func New(registry prometheusclient.Registerer, frontier Frontier) *FrontierMetrics {
	metrics := &FrontierMetrics{
		frontier: frontier,
		pendingPageVisits: prometheusclient.NewDesc(
			"yacycrawler_pending_page_visits",
			"Page visits pending on the frontier, leased or not.",
			nil, nil,
		),
	}
	registry.MustRegister(metrics)
	return metrics
}

func (metrics *FrontierMetrics) Describe(descriptions chan<- *prometheusclient.Desc) {
	descriptions <- metrics.pendingPageVisits
}

func (metrics *FrontierMetrics) Collect(samples chan<- prometheusclient.Metric) {
	amount, counted := metrics.frontier.AmountPending(context.Background())
	if !counted {
		return
	}
	samples <- prometheusclient.MustNewConstMetric(
		metrics.pendingPageVisits, prometheusclient.GaugeValue, float64(amount),
	)
}
