package prometheus_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	searchprogressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchprogressobservers/prometheus"
)

const secretQuery = "the query of the caller"

func exposition(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), "GET", "/metrics", nil)
	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(recorder, request)
	return recorder.Body.String()
}

func TestSearchProgressMetricsCountsEveryFactItIsTold(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := searchprogressobserversprometheus.New(registry)

	metrics.SearchServed(ctx, secretQuery, 3)
	metrics.SearchFailed(ctx, secretQuery, errors.New("engine away"))

	body := exposition(t, registry)
	for _, wanted := range []string{
		"webresearchmcp_searches_served_total 1",
		"webresearchmcp_search_failures_total 1",
	} {
		if !strings.Contains(body, wanted) {
			t.Errorf("metrics output missing %q, got:\n%s", wanted, body)
		}
	}
}

func TestSearchProgressMetricsExposesNoQuery(t *testing.T) {
	ctx := context.Background()
	registry := prometheusclient.NewRegistry()
	metrics := searchprogressobserversprometheus.New(registry)

	metrics.SearchServed(ctx, secretQuery, 1)
	metrics.SearchFailed(ctx, secretQuery, errors.New("engine away"))

	if body := exposition(t, registry); strings.Contains(body, secretQuery) {
		t.Errorf("metrics output carries the query of the caller, got:\n%s", body)
	}
}
