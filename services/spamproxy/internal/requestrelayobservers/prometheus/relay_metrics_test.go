package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelayobservers/prometheus"
)

func TestIncompleteResponsesAndClosedRequestsAreCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address, err := canonicalurl.CanonicalURLOf("http://a.example/")
	if err != nil {
		t.Fatal(err)
	}

	metrics.ResponseLeftIncomplete(t.Context(), address, requestrelay.RelayIdleTimeout,
		errors.New("upstream response stalled"))
	metrics.ClientClosedRequest(t.Context(), address)

	want := `
# HELP spamproxy_incomplete_responses_total Responses that ended before the whole body, by cause.
# TYPE spamproxy_incomplete_responses_total counter
spamproxy_incomplete_responses_total{cause="relay_idle_timeout"} 1
# HELP spamproxy_client_closed_requests_total Requests that the client closed before the response headers.
# TYPE spamproxy_client_closed_requests_total counter
spamproxy_client_closed_requests_total 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_incomplete_responses_total",
		"spamproxy_client_closed_requests_total"); err != nil {
		t.Fatal(err)
	}
}
