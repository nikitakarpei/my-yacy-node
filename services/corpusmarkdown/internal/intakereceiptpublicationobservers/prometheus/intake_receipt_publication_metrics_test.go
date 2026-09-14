package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	intakereceiptpublicationmetrics "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceiptpublicationobservers/prometheus"
)

const (
	pageURL = "https://example.org/a"
	subject = "yacy.scrape.pages.kept.abc"
)

func TestIntakeReceiptPublicationMetricsCountsEveryOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := intakereceiptpublicationmetrics.New(registry)
	ctx := context.Background()
	page := canonicalurltest.CanonicalURLOf(t, pageURL)
	cause := errors.New("no listener")

	metrics.IntakeReceiptSent(ctx, page, subject)
	metrics.IntakeReceiptEncodingFailed(ctx, page, cause)
	metrics.IntakeReceiptPublishingFailed(ctx, page, subject, cause)
	metrics.IntakeReceiptConfirmationFailed(ctx, page, subject, cause)

	req := httptest.NewRequestWithContext(ctx, "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		`corpusmarkdown_intake_receipt_publications_total{outcome="sent"} 1`,
		`corpusmarkdown_intake_receipt_publications_total{outcome="encoding_failed"} 1`,
		`corpusmarkdown_intake_receipt_publications_total{outcome="publishing_failed"} 1`,
		`corpusmarkdown_intake_receipt_publications_total{outcome="confirmation_failed"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
