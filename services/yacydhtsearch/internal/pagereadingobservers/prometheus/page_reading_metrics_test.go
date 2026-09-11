package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	pagereadingobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereadingobservers/prometheus"
)

const pageReadBudget = 3 * time.Second

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestOnePageReadingPublishesThePagesByOutcomeAndHowLongItTook(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := pagereadingobserversprometheus.New(registry, pageReadBudget)

	metrics.PageReadingPerformed(t.Context(), pagereading.PerformedPageReading{
		AmountOfPagesToRead:              16,
		AmountOfPagesRead:                4,
		AmountOfPagesUnreachable:         5,
		AmountOfPagesRefused:             3,
		AmountOfPagesUnreadable:          2,
		AmountOfPagesOfAnUnsupportedKind: 1,
		AmountOfPagesOutOfBudget:         1,
		TimeSpent:                        250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_page_reading_pages_total{outcome="read"} 4`,
		`yacydhtsearch_page_reading_pages_total{outcome="unreachable"} 5`,
		`yacydhtsearch_page_reading_pages_total{outcome="refused"} 3`,
		`yacydhtsearch_page_reading_pages_total{outcome="unreadable"} 2`,
		`yacydhtsearch_page_reading_pages_total{outcome="unsupported kind"} 1`,
		`yacydhtsearch_page_reading_pages_total{outcome="out of budget"} 1`,
		"yacydhtsearch_page_reading_duration_seconds_sum 0.25",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
