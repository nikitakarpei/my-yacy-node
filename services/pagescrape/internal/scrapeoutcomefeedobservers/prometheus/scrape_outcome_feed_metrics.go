// Package prometheus counts what the page feed carried to the callers waiting on a page,
// so an operator can tell a scrape failure that reached the feed from one that never left
// this process, and either of those from an intake receipt the feed could not carry over.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome = "outcome"

	outcomeAnnounced          = "announced"
	outcomeEncodingFailed     = "encoding_failed"
	outcomePublishingFailed   = "publishing_failed"
	outcomeConfirmationFailed = "confirmation_failed"

	outcomeCarried           = "carried"
	outcomeSubjectUnreadable = "subject_unreadable"
	outcomeNotCarried        = "not_carried"
)

var (
	scrapeFailureAnnouncementOutcomes = []string{
		outcomeAnnounced,
		outcomeEncodingFailed,
		outcomePublishingFailed,
		outcomeConfirmationFailed,
	}
	intakeReceiptCarriageOutcomes = []string{
		outcomeCarried,
		outcomeSubjectUnreadable,
		outcomeNotCarried,
	}
)

type ScrapeOutcomeFeedMetrics struct {
	scrapeFailureAnnouncements *prometheus.CounterVec
	intakeReceiptCarriages     *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *ScrapeOutcomeFeedMetrics {
	metrics := &ScrapeOutcomeFeedMetrics{
		scrapeFailureAnnouncements: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pagescrape_scrape_failure_announcements_total",
			Help: "Scrape failures announced on the page feed, by outcome.",
		}, []string{labelOutcome}),
		intakeReceiptCarriages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pagescrape_intake_receipt_carriages_total",
			Help: "Intake receipts carried from a corpus onto the page feed, by outcome.",
		}, []string{labelOutcome}),
	}
	for _, outcome := range scrapeFailureAnnouncementOutcomes {
		metrics.scrapeFailureAnnouncements.WithLabelValues(outcome)
	}
	for _, outcome := range intakeReceiptCarriageOutcomes {
		metrics.intakeReceiptCarriages.WithLabelValues(outcome)
	}
	registry.MustRegister(
		metrics.scrapeFailureAnnouncements,
		metrics.intakeReceiptCarriages,
	)
	return metrics
}

func (m *ScrapeOutcomeFeedMetrics) ScrapeFailureAnnounced(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
) {
	m.scrapeFailureAnnouncements.WithLabelValues(outcomeAnnounced).Inc()
}

func (m *ScrapeOutcomeFeedMetrics) ScrapeFailureEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.scrapeFailureAnnouncements.WithLabelValues(outcomeEncodingFailed).Inc()
}

func (m *ScrapeOutcomeFeedMetrics) ScrapeFailurePublishingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
	_ error,
) {
	m.scrapeFailureAnnouncements.WithLabelValues(outcomePublishingFailed).Inc()
}

func (m *ScrapeOutcomeFeedMetrics) ScrapeFailureConfirmationFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
	_ error,
) {
	m.scrapeFailureAnnouncements.WithLabelValues(outcomeConfirmationFailed).Inc()
}

func (m *ScrapeOutcomeFeedMetrics) IntakeReceiptCarried(
	_ context.Context,
	_ string,
	_ string,
) {
	m.intakeReceiptCarriages.WithLabelValues(outcomeCarried).Inc()
}

func (m *ScrapeOutcomeFeedMetrics) IntakeReceiptSubjectUnreadable(
	_ context.Context,
	_ string,
	_ error,
) {
	m.intakeReceiptCarriages.WithLabelValues(outcomeSubjectUnreadable).Inc()
}

func (m *ScrapeOutcomeFeedMetrics) IntakeReceiptNotCarried(
	_ context.Context,
	_ string,
	_ string,
	_ error,
) {
	m.intakeReceiptCarriages.WithLabelValues(outcomeNotCarried).Inc()
}
