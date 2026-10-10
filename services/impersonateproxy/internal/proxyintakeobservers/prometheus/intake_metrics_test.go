package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintake"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintakeobservers/prometheus"
)

func TestRefusedRequestsAndIncompleteResponsesAreCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)

	metrics.MethodRefused(t.Context(), "POST")
	metrics.TargetRefused(t.Context(), "/page")
	metrics.TargetRefused(t.Context(), "ftp://site.example/")
	metrics.ResponseLeftIncomplete(t.Context(), "http://site.example/",
		proxyintake.PageReadFailed, errors.New("origin stopped"))

	want := `
# HELP impersonateproxy_incomplete_responses_total Responses that ended before the whole body, by cause.
# TYPE impersonateproxy_incomplete_responses_total counter
impersonateproxy_incomplete_responses_total{cause="page_read_failed"} 1
# HELP impersonateproxy_refused_requests_total Requests that the proxy refused, by the part it refused.
# TYPE impersonateproxy_refused_requests_total counter
impersonateproxy_refused_requests_total{part="method"} 1
impersonateproxy_refused_requests_total{part="target"} 2
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want)); err != nil {
		t.Fatal(err)
	}
}
