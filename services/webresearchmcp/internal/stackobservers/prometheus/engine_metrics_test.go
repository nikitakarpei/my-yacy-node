package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	stackobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/stackobservers/prometheus"
)

const metricName = "webresearchmcp_search_engine_failures_total"

func TestEngineMetricsCountEveryFailureByEngine(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := stackobserversprometheus.New(registry)

	metrics.EngineFailed(context.Background(), "yacy", "subject", errors.New("away"))
	metrics.EngineFailed(context.Background(), "yacy", "subject", errors.New("away"))
	metrics.EngineFailed(context.Background(), "corpustext", "subject", errors.New("away"))

	expected := `
# HELP webresearchmcp_search_engine_failures_total ` +
		`Searches a search engine of the stack did not answer, by engine.
# TYPE webresearchmcp_search_engine_failures_total counter
webresearchmcp_search_engine_failures_total{engine="corpustext"} 1
webresearchmcp_search_engine_failures_total{engine="yacy"} 2
`
	if err := testutil.GatherAndCompare(
		registry, strings.NewReader(expected), metricName,
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
