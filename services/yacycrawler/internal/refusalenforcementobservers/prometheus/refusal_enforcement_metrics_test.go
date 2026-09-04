package prometheus_test

import (
	"context"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	refusalenforcementobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/refusalenforcementobservers/prometheus"
)

const metricName = "yacycrawler_link_discovery_refusals_enforced_total"

func TestRefusalEnforcementMetricsCountEveryEnforcedLinkDiscoveryRefusal(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := refusalenforcementobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://host.example/page")

	metrics.LinkDiscoveryRefusalEnforced(context.Background(), pageURL)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+metricName+
			" Pages whose refusal of link discovery the crawler enforced.\n"+
			"# TYPE "+metricName+" counter\n"+
			metricName+" 1\n",
	), metricName); err != nil {
		t.Fatal(err)
	}
}

func TestAnEnforcedLinkDiscoveryRefusalReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	refusalenforcementobserversprometheus.New(registry)

	if counters := testutil.CollectAndCount(registry, metricName); counters != 1 {
		t.Fatalf("counters = %d, want 1", counters)
	}
}
