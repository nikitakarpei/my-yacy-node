package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	robotsrulesobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrulesobservers/prometheus"
)

const (
	lookupsMetricName = "yacycrawler_robots_rules_lookups_total"
	notReadMetricName = "yacycrawler_robots_rules_not_read_total"
	notKeptMetricName = "yacycrawler_robots_rules_not_kept_total"
)

func TestRobotsRulesMetricsCountLookupsAndFailures(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := robotsrulesobserversprometheus.New(registry)
	robotsTxtURL := canonicalurltest.CanonicalURLOf(t, "http://host.example/robots.txt")
	ctx := context.Background()

	metrics.RobotsRulesMissed(ctx, robotsTxtURL)
	metrics.RobotsRulesFound(ctx, robotsTxtURL)
	metrics.RobotsRulesFound(ctx, robotsTxtURL)
	metrics.RobotsRulesNotRead(ctx, robotsTxtURL, errors.New("down"))
	metrics.RobotsRulesNotKept(ctx, robotsTxtURL, errors.New("down"))

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+lookupsMetricName+
			" Lookups of the kept robots rules of a site, by whether the rules were kept.\n"+
			"# TYPE "+lookupsMetricName+" counter\n"+
			lookupsMetricName+`{result="found"} 2`+"\n"+
			lookupsMetricName+`{result="missed"} 1`+"\n"+
			"# HELP "+notKeptMetricName+
			" Robots rules the crawler could not keep for the other pages of their site.\n"+
			"# TYPE "+notKeptMetricName+" counter\n"+
			notKeptMetricName+" 1\n"+
			"# HELP "+notReadMetricName+
			" Kept robots rules the crawler could not read, so it fetched robots.txt again.\n"+
			"# TYPE "+notReadMetricName+" counter\n"+
			notReadMetricName+" 1\n",
	), lookupsMetricName, notReadMetricName, notKeptMetricName); err != nil {
		t.Fatal(err)
	}
}
