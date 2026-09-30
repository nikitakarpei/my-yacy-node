package prometheus_test

import (
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/failurereporting"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/failurereporting/prometheus"
)

func TestFailedUpstreamRequestsAreCountedByStep(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)

	metrics.RoundTripFailed(t.Context(), "http://a.example/", failurereporting.ConnectFailed,
		errors.New("connection refused"))

	want := `
# HELP spamproxy_upstream_request_failures_total Upstream requests that failed, by the step that failed.
# TYPE spamproxy_upstream_request_failures_total counter
spamproxy_upstream_request_failures_total{step="connect_failed"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_upstream_request_failures_total"); err != nil {
		t.Fatal(err)
	}
}
