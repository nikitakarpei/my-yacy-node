// Package prometheus counts the pages one query read by outcome, and reports
// how long the reading of those pages took.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

const (
	labelOutcome               = "outcome"
	outcomePageRead            = "read"
	outcomePageUnreachable     = "unreachable"
	outcomePageRefused         = "refused"
	outcomePageUnreadable      = "unreadable"
	outcomePageUnsupportedKind = "unsupported kind"
	outcomePageOutOfBudget     = "out of budget"
	durationBuckets            = 12
	budgetShare                = 1024
	overThePageReadBudget      = 2
)

type PageReadingMetrics struct {
	pages                      *prometheusclient.CounterVec
	pageReadingDurationSeconds prometheusclient.Histogram
}

func New(
	registry prometheusclient.Registerer,
	pageReadBudget time.Duration,
) *PageReadingMetrics {
	metrics := &PageReadingMetrics{
		pages: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_page_reading_pages_total",
			Help: "Pages one query read, by outcome.",
		}, []string{labelOutcome}),
		pageReadingDurationSeconds: prometheusclient.NewHistogram(
			prometheusclient.HistogramOpts{
				Name: "yacydhtsearch_page_reading_duration_seconds",
				Help: "Time the reading of the pages of one query took, in seconds.",
				Buckets: prometheusclient.ExponentialBucketsRange(
					pageReadBudget.Seconds()/budgetShare,
					pageReadBudget.Seconds()*overThePageReadBudget,
					durationBuckets,
				),
			},
		),
	}
	registry.MustRegister(
		metrics.pages,
		metrics.pageReadingDurationSeconds,
	)

	return metrics
}

func (m *PageReadingMetrics) PageReadingPerformed(
	_ context.Context,
	pageReading pagereading.PerformedPageReading,
) {
	m.pageReadingDurationSeconds.Observe(pageReading.TimeSpent.Seconds())
	for outcome, amountOfPages := range map[string]int{
		outcomePageRead:            pageReading.AmountOfPagesRead,
		outcomePageUnreachable:     pageReading.AmountOfPagesUnreachable,
		outcomePageRefused:         pageReading.AmountOfPagesRefused,
		outcomePageUnreadable:      pageReading.AmountOfPagesUnreadable,
		outcomePageUnsupportedKind: pageReading.AmountOfPagesOfAnUnsupportedKind,
		outcomePageOutOfBudget:     pageReading.AmountOfPagesOutOfBudget,
	} {
		m.pages.WithLabelValues(outcome).Add(float64(amountOfPages))
	}
}
