// Package prometheus counts the facts of adding classifications to pages as
// Prometheus metrics.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/roundtrippers/classificationadding"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

var (
	scoreBuckets           = prometheusclient.LinearBuckets(0.1, 0.1, 10)
	slotWaitSecondsBuckets = []float64{
		0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	}
)

type ClassificationMetrics struct {
	classifiedPages        prometheusclient.Counter
	verdicts               *prometheusclient.CounterVec
	scores                 *prometheusclient.HistogramVec
	readingSlotWaitSeconds prometheusclient.Histogram
	skippedClassifications *prometheusclient.CounterVec
	refusedPages           *prometheusclient.CounterVec
	bodyPrefixReadFailures prometheusclient.Counter
}

func New(registry prometheusclient.Registerer) *ClassificationMetrics {
	metrics := &ClassificationMetrics{
		classifiedPages: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "classifierproxy_classified_pages_total",
			Help: "Classified pages.",
		}),
		verdicts: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "classifierproxy_verdicts_total",
			Help: "Verdicts on the classified pages, by category and verdict.",
		}, []string{"category", "verdict"}),
		scores: prometheusclient.NewHistogramVec(prometheusclient.HistogramOpts{
			Name:    "classifierproxy_scores",
			Help:    "Scores of the classified pages, by category.",
			Buckets: scoreBuckets,
		}, []string{"category"}),
		readingSlotWaitSeconds: prometheusclient.NewHistogram(prometheusclient.HistogramOpts{
			Name:    "classifierproxy_reading_slot_wait_duration_seconds",
			Help:    "Time that a page waits for a place among the pages read at once.",
			Buckets: slotWaitSecondsBuckets,
		}),
		skippedClassifications: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "classifierproxy_skipped_classifications_total",
			Help: "Upstream responses relayed without a classification.",
		}, []string{"reason"}),
		refusedPages: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "classifierproxy_refused_pages_total",
			Help: "Pages that the proxy answers with its own error status, by reason.",
		}, []string{"reason"}),
		bodyPrefixReadFailures: prometheusclient.NewCounter(prometheusclient.CounterOpts{
			Name: "classifierproxy_body_prefix_read_failures_total",
			Help: "Pages whose body failed before the part that the proxy classifies.",
		}),
	}
	registry.MustRegister(
		metrics.classifiedPages,
		metrics.verdicts,
		metrics.scores,
		metrics.readingSlotWaitSeconds,
		metrics.skippedClassifications,
		metrics.refusedPages,
		metrics.bodyPrefixReadFailures,
	)
	return metrics
}

func (m *ClassificationMetrics) PageClassified(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	classification pageclassification.Classification,
) {
	m.classifiedPages.Inc()
	for category, assessment := range classification {
		m.verdicts.WithLabelValues(string(category), assessment.Verdict().String()).Inc()
		m.scores.WithLabelValues(string(category)).Observe(assessment.Score)
	}
}

func (m *ClassificationMetrics) ReadingSlotWaited(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	m.readingSlotWaitSeconds.Observe(slotWait.Seconds())
}

func (m *ClassificationMetrics) ClassificationSkipped(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason classificationadding.SkipReason,
) {
	m.skippedClassifications.WithLabelValues(string(reason)).Inc()
}

func (m *ClassificationMetrics) PageRefused(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	reason classificationadding.RefusalReason,
) {
	m.refusedPages.WithLabelValues(string(reason)).Inc()
}

func (m *ClassificationMetrics) BodyPrefixReadFailed(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	m.bodyPrefixReadFailures.Inc()
}
