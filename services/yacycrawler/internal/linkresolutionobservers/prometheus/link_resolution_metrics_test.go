package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	linkresolutionobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/linkresolutionobservers/prometheus"
)

const (
	baseHrefMetricName  = "yacycrawler_page_base_hrefs_unresolved_total"
	linkHrefsMetricName = "yacycrawler_page_link_hrefs_unresolved_total"
)

func TestLinkResolutionMetricsCountEveryHrefThatCannotBeResolved(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := linkresolutionobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://host.example/page")

	metrics.BaseHrefUnresolved(context.Background(), pageURL, "::base::", errors.New("parse"))
	metrics.LinkHrefsUnresolved(context.Background(), pageURL, 3)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+baseHrefMetricName+
			" Pages whose base href cannot be resolved, read against the page url.\n"+
			"# TYPE "+baseHrefMetricName+" counter\n"+
			baseHrefMetricName+" 1\n"+
			"# HELP "+linkHrefsMetricName+
			" Link hrefs that cannot be resolved, left off the frontier.\n"+
			"# TYPE "+linkHrefsMetricName+" counter\n"+
			linkHrefsMetricName+" 3\n",
	), baseHrefMetricName, linkHrefsMetricName); err != nil {
		t.Fatal(err)
	}
}

func TestEveryUnresolvedHrefReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	linkresolutionobserversprometheus.New(registry)

	if counters := testutil.CollectAndCount(
		registry, baseHrefMetricName, linkHrefsMetricName,
	); counters != 2 {
		t.Fatalf("counters = %d, want 2", counters)
	}
}

func TestLinkResolutionMetricsExposeNoPageAddress(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := linkresolutionobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://private.example/page")

	metrics.BaseHrefUnresolved(context.Background(), pageURL, "::base::", errors.New("parse"))

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			if len(metric.GetLabel()) != 0 {
				t.Fatalf("%s carries labels %v", family.GetName(), metric.GetLabel())
			}
		}
	}
}
