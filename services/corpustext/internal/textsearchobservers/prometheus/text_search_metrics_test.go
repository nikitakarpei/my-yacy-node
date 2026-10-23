package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	textsearchmetrics "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearchobservers/prometheus"
)

func TestTextSearchMetricsCountSearchesByOutcomeAndTimeThem(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := textsearchmetrics.New(registry)
	ctx := context.Background()
	search := textsearch.Search{Query: "words", Limit: textsearch.DefaultMatchLimit}

	metrics.SearchAnswered(ctx, search, 20*time.Millisecond, 3)
	metrics.SearchAnswered(ctx, search, 30*time.Millisecond, 0)
	metrics.SearchFailed(ctx, search, time.Second, errors.New("index down"))

	body := exposition(t, registry)
	for _, want := range []string{
		`corpustext_searches_total{outcome="answered"} 2`,
		`corpustext_searches_total{outcome="failed"} 1`,
		"corpustext_search_duration_seconds_count 3",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}
}

func exposition(t *testing.T, registry *prometheus.Registry) string {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	response := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(response, request)

	return response.Body.String()
}
