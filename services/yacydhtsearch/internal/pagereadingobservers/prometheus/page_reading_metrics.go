// Package prometheus counts the pages one query read by outcome, and reports
// how long the reading of those pages took.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

const (
	labelOutcome               = "outcome"
	labelActivity              = "activity"
	activityFetching           = "fetching"
	activityReading            = "reading"
	outcomePageRead            = "read"
	outcomePageUnreachable     = "unreachable"
	outcomePageRefused         = "refused"
	outcomePageGone            = "gone"
	outcomePageUnreadable      = "unreadable"
	outcomePageUnsupportedKind = "unsupported kind"
	outcomePageOutOfBudget     = "out of budget"
)

type PageReadingMetrics struct {
	pagesRead                  prometheusclient.Counter
	pagesUnreachable           prometheusclient.Counter
	pagesRefused               prometheusclient.Counter
	pagesGone                  prometheusclient.Counter
	pagesUnreadable            prometheusclient.Counter
	pagesOfAnUnsupportedKind   prometheusclient.Counter
	pagesOutOfBudget           prometheusclient.Counter
	pageReadingDurationSeconds prometheusclient.Histogram
	timeSpentFetchingSeconds   prometheusclient.Counter
	timeSpentReadingSeconds    prometheusclient.Counter
}

func New(
	registry prometheusclient.Registerer,
	pageReadBudget time.Duration,
) *PageReadingMetrics {
	pages := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_page_reading_pages_total",
		Help: "Pages one query read, by outcome.",
	}, []string{labelOutcome})
	pageReadingDurationSeconds := prometheusclient.NewHistogram(
		prometheusclient.HistogramOpts{
			Name:    "yacydhtsearch_page_reading_duration_seconds",
			Help:    "Time the reading of the pages of one query took, in seconds.",
			Buckets: budgetbuckets.DurationBucketsFor(pageReadBudget),
		},
	)
	timeSpentSeconds := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_page_reading_time_spent_seconds_total",
		Help: "Time the pages of all queries spent, by activity, in seconds.",
	}, []string{labelActivity})
	registry.MustRegister(pages, pageReadingDurationSeconds, timeSpentSeconds)

	return &PageReadingMetrics{
		pagesRead:                  pages.WithLabelValues(outcomePageRead),
		pagesUnreachable:           pages.WithLabelValues(outcomePageUnreachable),
		pagesRefused:               pages.WithLabelValues(outcomePageRefused),
		pagesGone:                  pages.WithLabelValues(outcomePageGone),
		pagesUnreadable:            pages.WithLabelValues(outcomePageUnreadable),
		pagesOfAnUnsupportedKind:   pages.WithLabelValues(outcomePageUnsupportedKind),
		pagesOutOfBudget:           pages.WithLabelValues(outcomePageOutOfBudget),
		pageReadingDurationSeconds: pageReadingDurationSeconds,
		timeSpentFetchingSeconds:   timeSpentSeconds.WithLabelValues(activityFetching),
		timeSpentReadingSeconds:    timeSpentSeconds.WithLabelValues(activityReading),
	}
}

func (m *PageReadingMetrics) PageReadingPerformed(
	_ context.Context,
	pageReading pagereading.PerformedPageReading,
) {
	m.pageReadingDurationSeconds.Observe(pageReading.TimeSpent.Seconds())
	m.timeSpentFetchingSeconds.Add(pageReading.TimeSpentFetching.Seconds())
	m.timeSpentReadingSeconds.Add(pageReading.TimeSpentReading.Seconds())
	m.pagesRead.Add(float64(pageReading.AmountOfPagesRead))
	m.pagesUnreachable.Add(float64(pageReading.AmountOfPagesUnreachable))
	m.pagesRefused.Add(float64(pageReading.AmountOfPagesRefused))
	m.pagesGone.Add(float64(pageReading.AmountOfPagesGone))
	m.pagesUnreadable.Add(float64(pageReading.AmountOfPagesUnreadable))
	m.pagesOfAnUnsupportedKind.Add(float64(pageReading.AmountOfPagesOfAnUnsupportedKind))
	m.pagesOutOfBudget.Add(float64(pageReading.AmountOfPagesOutOfBudget))
}
