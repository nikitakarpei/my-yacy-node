package prometheus

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	labelScrapeRequestDisposal = "disposal"
	labelRetryCause            = "cause"
)

const (
	disposalOffered    = "offered"
	disposalScheduled  = "scheduled"
	disposalUnreadable = "unreadable"
)

var scrapeRequestDisposals = []string{
	disposalOffered,
	disposalScheduled,
	disposalUnreadable,
	string(pagescrapecontract.NotModified),
	string(pagescrapecontract.AccessRefused),
	string(pagescrapecontract.RedirectsExhausted),
	string(pagescrapecontract.RedirectTargetInvalid),
	string(pagescrapecontract.Oversized),
	string(pagescrapecontract.NoReasonGiven),
	string(pagescrapecontract.Deferred),
	string(pagescrapecontract.DeferredTooLong),
}

const (
	retryCauseOfferRefused    = "offer-refused"
	retryCauseScheduleRefused = "schedule-refused"
)

var retryCauses = []string{
	retryCauseOfferRefused,
	retryCauseScheduleRefused,
}

type ScrapeIntakeMetrics struct {
	scrapeRequestsDisposed     *prometheus.CounterVec
	scrapeRequestsLeftForRetry *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *ScrapeIntakeMetrics {
	scrapeRequestsDisposed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pagescrape_scrape_requests_disposed_total",
		Help: "Scrape requests, by the disposal that ended their intake.",
	}, []string{labelScrapeRequestDisposal})
	for _, disposal := range scrapeRequestDisposals {
		scrapeRequestsDisposed.WithLabelValues(disposal)
	}
	scrapeRequestsLeftForRetry := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pagescrape_scrape_requests_left_for_retry_total",
		Help: "Scrape requests the service could not finish yet, by cause.",
	}, []string{labelRetryCause})
	for _, cause := range retryCauses {
		scrapeRequestsLeftForRetry.WithLabelValues(cause)
	}
	registry.MustRegister(
		scrapeRequestsDisposed,
		scrapeRequestsLeftForRetry,
	)
	return &ScrapeIntakeMetrics{
		scrapeRequestsDisposed:     scrapeRequestsDisposed,
		scrapeRequestsLeftForRetry: scrapeRequestsLeftForRetry,
	}
}

func (m *ScrapeIntakeMetrics) ScrapeRequestInvalid(_ context.Context, _ string, _ error) {
	m.dispose(disposalUnreadable)
}

func (m *ScrapeIntakeMetrics) ScrapeRequestReceived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
}

func (m *ScrapeIntakeMetrics) OriginFetchFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (m *ScrapeIntakeMetrics) PageOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalOffered)
}

func (m *ScrapeIntakeMetrics) PageNotOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.leaveForRetry(retryCauseOfferRefused)
}

func (m *ScrapeIntakeMetrics) ScrapeDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	m.dispose(disposalScheduled)
}

func (m *ScrapeIntakeMetrics) ScrapeScheduleFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.leaveForRetry(retryCauseScheduleRefused)
}

func (m *ScrapeIntakeMetrics) ScrapeFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	m.dispose(string(reason))
}

func (m *ScrapeIntakeMetrics) dispose(disposal string) {
	m.scrapeRequestsDisposed.WithLabelValues(disposal).Inc()
}

func (m *ScrapeIntakeMetrics) leaveForRetry(cause string) {
	m.scrapeRequestsLeftForRetry.WithLabelValues(cause).Inc()
}
