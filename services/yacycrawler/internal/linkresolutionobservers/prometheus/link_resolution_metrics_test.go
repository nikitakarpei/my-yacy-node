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
	baseURLMetricName = "yacycrawler_page_base_urls_unresolved_total"
	linksMetricName   = "yacycrawler_page_links_unresolved_total"
)

func TestLinkResolutionMetricsCountEveryLinkThatCannotBeResolved(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := linkresolutionobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://host.example/page")

	metrics.BaseURLUnresolved(context.Background(), pageURL, "::base::", errors.New("parse"))
	metrics.LinksUnresolved(context.Background(), pageURL, 3)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+baseURLMetricName+
			" Pages whose stated base URL cannot be resolved, read against the page URL instead.\n"+
			"# TYPE "+baseURLMetricName+" counter\n"+
			baseURLMetricName+" 1\n"+
			"# HELP "+linksMetricName+
			" Links that cannot be resolved into a URL, left off the frontier.\n"+
			"# TYPE "+linksMetricName+" counter\n"+
			linksMetricName+" 3\n",
	), baseURLMetricName, linksMetricName); err != nil {
		t.Fatal(err)
	}
}

func TestEveryUnresolvedLinkReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	linkresolutionobserversprometheus.New(registry)

	if counters := testutil.CollectAndCount(
		registry, baseURLMetricName, linksMetricName,
	); counters != 2 {
		t.Fatalf("counters = %d, want 2", counters)
	}
}

func TestLinkResolutionMetricsExposeNoPageAddress(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := linkresolutionobserversprometheus.New(registry)
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://private.example/page")

	metrics.BaseURLUnresolved(context.Background(), pageURL, "::base::", errors.New("parse"))

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
