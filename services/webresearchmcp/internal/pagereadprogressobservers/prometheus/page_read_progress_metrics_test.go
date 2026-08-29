package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
	pagereadprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pagereadprogressobservers/prometheus"
)

const unexposedPageAddress = "https://example.test/private-page"

func exposition(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(recorder, request)
	return recorder.Body.String()
}

func TestPageReadProgressMetricsCountsEveryFactItIsTold(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := pagereadprogressobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, unexposedPageAddress)
	cause := errors.New("told alongside the fact")

	metrics.PageAnswered(ctx, pageURL, pageread.PageFetched)
	metrics.PageAnswered(ctx, pageURL, pageread.FetchNotNeeded)
	metrics.MarkdownRecallFailed(ctx, pageURL, cause)
	metrics.ScrapeOutcomeListenFailed(ctx, pageURL, cause)
	metrics.ScrapeRequestFailed(ctx, pageURL, cause)
	metrics.FetchOutcomeNotHeard(ctx, pageURL, time.Second, cause)

	body := exposition(t, registry)
	for _, wanted := range []string{
		`webresearchmcp_pages_answered_total{fetch_outcome="page-fetched"} 1`,
		`webresearchmcp_pages_answered_total{fetch_outcome="fetch-not-needed"} 1`,
		"webresearchmcp_markdown_recall_failures_total 1",
		"webresearchmcp_scrape_outcome_listen_failures_total 1",
		"webresearchmcp_scrape_request_failures_total 1",
		"webresearchmcp_fetch_outcomes_not_heard_total 1",
	} {
		if !strings.Contains(body, wanted) {
			t.Errorf("metrics output missing %q, got:\n%s", wanted, body)
		}
	}
}

func TestPageReadProgressMetricsExposesNoPageAddress(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := pagereadprogressobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, unexposedPageAddress)

	metrics.PageAnswered(ctx, pageURL, pageread.PageFetched)
	metrics.MarkdownRecallFailed(ctx, pageURL, errors.New("corpus away"))

	if body := exposition(t, registry); strings.Contains(body, "example.test") {
		t.Errorf("metrics output labels a counter by the page address, got:\n%s", body)
	}
}
