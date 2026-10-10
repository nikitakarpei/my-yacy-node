package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/pagefetchersobservers/tlsclient/prometheus"
)

func TestPageFetchesAreCountedByOutcome(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)

	metrics.PageFetched(t.Context(), "https://site.example/", 200)
	metrics.PageFetched(t.Context(), "https://site.example/gone", 404)
	metrics.FetchFailed(t.Context(), "https://site.example/", errors.New("tunnel refused"))
	metrics.FetchCancelled(t.Context(), "https://site.example/")

	want := `
# HELP impersonateproxy_cancelled_fetches_total Page fetches that the client closed before the response.
# TYPE impersonateproxy_cancelled_fetches_total counter
impersonateproxy_cancelled_fetches_total 1
# HELP impersonateproxy_failed_fetches_total Page fetches that ended without a response.
# TYPE impersonateproxy_failed_fetches_total counter
impersonateproxy_failed_fetches_total 1
# HELP impersonateproxy_fetched_pages_total Pages that the origin answered, by response status.
# TYPE impersonateproxy_fetched_pages_total counter
impersonateproxy_fetched_pages_total{status="200"} 1
impersonateproxy_fetched_pages_total{status="404"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want)); err != nil {
		t.Fatal(err)
	}
}
