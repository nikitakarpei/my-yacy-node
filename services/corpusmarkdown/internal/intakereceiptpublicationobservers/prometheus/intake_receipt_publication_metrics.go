// Package prometheus counts how every intake receipt the corpus writes back fared, so an
// operator can tell a receipt that reached the caller from one that never left this
// process, and either of those from one the broker never confirmed.
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	labelOutcome = "outcome"

	outcomeSent               = "sent"
	outcomeEncodingFailed     = "encoding_failed"
	outcomePublishingFailed   = "publishing_failed"
	outcomeConfirmationFailed = "confirmation_failed"
)

var intakeReceiptPublicationOutcomes = []string{
	outcomeSent,
	outcomeEncodingFailed,
	outcomePublishingFailed,
	outcomeConfirmationFailed,
}

type IntakeReceiptPublicationMetrics struct {
	intakeReceiptPublications *prometheus.CounterVec
}

func New(registry prometheus.Registerer) *IntakeReceiptPublicationMetrics {
	metrics := &IntakeReceiptPublicationMetrics{
		intakeReceiptPublications: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "corpusmarkdown_intake_receipt_publications_total",
			Help: "Intake receipts written back to the waiting caller, by outcome.",
		}, []string{labelOutcome}),
	}
	for _, outcome := range intakeReceiptPublicationOutcomes {
		metrics.intakeReceiptPublications.WithLabelValues(outcome)
	}
	registry.MustRegister(metrics.intakeReceiptPublications)
	return metrics
}

func (m *IntakeReceiptPublicationMetrics) IntakeReceiptSent(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
) {
	m.count(outcomeSent)
}

func (m *IntakeReceiptPublicationMetrics) IntakeReceiptEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	m.count(outcomeEncodingFailed)
}

func (m *IntakeReceiptPublicationMetrics) IntakeReceiptPublishingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
	_ error,
) {
	m.count(outcomePublishingFailed)
}

func (m *IntakeReceiptPublicationMetrics) IntakeReceiptConfirmationFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
	_ error,
) {
	m.count(outcomeConfirmationFailed)
}

func (m *IntakeReceiptPublicationMetrics) count(outcome string) {
	m.intakeReceiptPublications.WithLabelValues(outcome).Inc()
}
