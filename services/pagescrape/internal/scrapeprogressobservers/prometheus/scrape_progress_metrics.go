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

	disposalOffered         = "offered"
	disposalScheduled       = "scheduled"
	disposalOfferRefused    = "offer-refused"
	disposalScheduleRefused = "schedule-refused"
	disposalUnreadable      = "unreadable"
)

var scrapeRequestDisposals = []string{
	disposalOffered,
	disposalScheduled,
	disposalOfferRefused,
	disposalScheduleRefused,
	disposalUnreadable,
	string(pagescrapecontract.NotModified),
	string(pagescrapecontract.AccessRefused),
	string(pagescrapecontract.LandedURLInvalid),
	string(pagescrapecontract.Oversized),
	string(pagescrapecontract.NoReasonGiven),
	string(pagescrapecontract.Deferred),
	string(pagescrapecontract.DeferredTooLong),
}

type ScrapeProgressMetrics struct {
	scrapeRequestsReceived prometheus.Counter
	scrapeRequestsDisposed *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *ScrapeProgressMetrics {
	scrapeRequestsReceived := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pagescrape_scrape_requests_received_total",
		Help: "Scrape requests received.",
	})
	scrapeRequestsDisposed := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pagescrape_scrape_requests_disposed_total",
		Help: "Scrape requests, by how the service disposed of each one.",
	}, []string{labelScrapeRequestDisposal})
	for _, disposal := range scrapeRequestDisposals {
		scrapeRequestsDisposed.WithLabelValues(disposal)
	}
	registry.MustRegister(
		scrapeRequestsReceived,
		scrapeRequestsDisposed,
	)
	return &ScrapeProgressMetrics{
		scrapeRequestsReceived: scrapeRequestsReceived,
		scrapeRequestsDisposed: scrapeRequestsDisposed,
	}
}

func (m *ScrapeProgressMetrics) ScrapeRequestInvalid(_ context.Context, _ string, _ error) {
	m.dispose(disposalUnreadable)
}

func (m *ScrapeProgressMetrics) ScrapeRequestReceived(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
) {
	m.scrapeRequestsReceived.Inc()
}

func (m *ScrapeProgressMetrics) OriginReadFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
}

func (m *ScrapeProgressMetrics) PageOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ canonicalurl.CanonicalURL,
) {
	m.dispose(disposalOffered)
}

func (m *ScrapeProgressMetrics) PageNotOffered(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.dispose(disposalOfferRefused)
}

func (m *ScrapeProgressMetrics) ScrapeDeferred(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	m.dispose(disposalScheduled)
}

func (m *ScrapeProgressMetrics) ScrapeScheduleFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.dispose(disposalScheduleRefused)
}

func (m *ScrapeProgressMetrics) ScrapeFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	m.dispose(string(reason))
}

func (m *ScrapeProgressMetrics) dispose(disposal string) {
	m.scrapeRequestsDisposed.WithLabelValues(disposal).Inc()
}
