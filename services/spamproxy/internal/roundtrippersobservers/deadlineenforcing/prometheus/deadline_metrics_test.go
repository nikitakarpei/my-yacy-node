package prometheus_test

import (
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/deadlineenforcing"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippersobservers/deadlineenforcing/prometheus"
)

func TestDeadlineRefusalsAreCountedByReason(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := prometheus.New(registry)
	address, err := canonicalurl.CanonicalURLOf("http://a.example/")
	if err != nil {
		t.Fatal(err)
	}

	metrics.RequestRefused(t.Context(), address, deadlineenforcing.HeadersDeadline)

	want := `
# HELP spamproxy_deadline_refusals_total Requests refused because their response headers could not arrive in time, by reason.
# TYPE spamproxy_deadline_refusals_total counter
spamproxy_deadline_refusals_total{reason="headers_deadline"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(want),
		"spamproxy_deadline_refusals_total"); err != nil {
		t.Fatal(err)
	}
}
