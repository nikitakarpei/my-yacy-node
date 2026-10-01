// Package prometheus counts the pages one query read by outcome, the pages read by
// spam verdict and the pages it read but did not want, and reports how long the
// query waited for the pages it wanted.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/budgetbuckets"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

const (
	labelOutcome                = "outcome"
	labelActivity               = "activity"
	labelSpamVerdict            = "spam_verdict"
	activityFetching            = "fetching"
	activityReading             = "reading"
	outcomePageRead             = "read"
	outcomePageUnreachable      = "unreachable"
	outcomePageRefused          = "refused"
	outcomePageGone             = "gone"
	outcomePageRefusingIndexing = "refusing indexing"
	outcomePageUnreadable       = "unreadable"
	outcomePageUnsupportedKind  = "unsupported kind"
	outcomePageOutOfBudget      = "out of budget"
	outcomePageCutOff           = "cut off"
)

var spamVerdicts = []spamassessment.Verdict{
	spamassessment.Unassessed,
	spamassessment.Clean,
	spamassessment.Spam,
}

type PageReadingMetrics struct {
	pagesRead                  prometheusclient.Counter
	pagesUnreachable           prometheusclient.Counter
	pagesRefused               prometheusclient.Counter
	pagesGone                  prometheusclient.Counter
	pagesRefusingIndexing      prometheusclient.Counter
	pagesUnreadable            prometheusclient.Counter
	pagesOfAnUnsupportedKind   prometheusclient.Counter
	pagesOutOfBudget           prometheusclient.Counter
	pagesCutOff                prometheusclient.Counter
	pagesUnwanted              prometheusclient.Counter
	pagesReadPerSpamVerdict    *prometheusclient.CounterVec
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
			Help:    "Time one query waited for the pages it wanted, in seconds.",
			Buckets: budgetbuckets.DurationBucketsFor(pageReadBudget),
		},
	)
	timeSpentSeconds := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_page_reading_time_spent_seconds_total",
		Help: "Time the pages of all queries spent, by activity, in seconds.",
	}, []string{labelActivity})
	pagesReadPerSpamVerdict := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_page_reading_pages_read_total",
		Help: "Pages one query read, by spam verdict.",
	}, []string{labelSpamVerdict})
	pagesUnwanted := prometheusclient.NewCounter(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_page_reading_pages_unwanted_total",
		Help: "Pages one query started to read but did not want.",
	})
	registry.MustRegister(
		pages,
		pagesUnwanted,
		pageReadingDurationSeconds,
		timeSpentSeconds,
		pagesReadPerSpamVerdict,
	)
	for _, spamVerdict := range spamVerdicts {
		pagesReadPerSpamVerdict.WithLabelValues(spamVerdict.String())
	}

	return &PageReadingMetrics{
		pagesRead:                  pages.WithLabelValues(outcomePageRead),
		pagesUnreachable:           pages.WithLabelValues(outcomePageUnreachable),
		pagesRefused:               pages.WithLabelValues(outcomePageRefused),
		pagesGone:                  pages.WithLabelValues(outcomePageGone),
		pagesRefusingIndexing:      pages.WithLabelValues(outcomePageRefusingIndexing),
		pagesUnreadable:            pages.WithLabelValues(outcomePageUnreadable),
		pagesOfAnUnsupportedKind:   pages.WithLabelValues(outcomePageUnsupportedKind),
		pagesOutOfBudget:           pages.WithLabelValues(outcomePageOutOfBudget),
		pagesCutOff:                pages.WithLabelValues(outcomePageCutOff),
		pagesUnwanted:              pagesUnwanted,
		pagesReadPerSpamVerdict:    pagesReadPerSpamVerdict,
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
	m.pagesRefusingIndexing.Add(float64(pageReading.AmountOfPagesRefusingIndexing))
	m.pagesUnreadable.Add(float64(pageReading.AmountOfPagesUnreadable))
	m.pagesOfAnUnsupportedKind.Add(float64(pageReading.AmountOfPagesOfAnUnsupportedKind))
	m.pagesOutOfBudget.Add(float64(pageReading.AmountOfPagesOutOfBudget))
	m.pagesCutOff.Add(float64(pageReading.AmountOfPagesCutOff))
	m.pagesUnwanted.Add(float64(pageReading.AmountOfPagesUnwanted))
	for spamVerdict, amountOfPagesRead := range pageReading.AmountOfPagesReadPerSpamVerdict {
		m.pagesReadPerSpamVerdict.WithLabelValues(spamVerdict.String()).
			Add(float64(amountOfPagesRead))
	}
}
