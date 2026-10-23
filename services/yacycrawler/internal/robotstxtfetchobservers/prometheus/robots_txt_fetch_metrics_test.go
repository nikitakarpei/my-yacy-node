package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	robotstxtfetchobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotstxtfetchobservers/prometheus"
)

const fetchesMetricName = "yacycrawler_robots_txt_fetches_total"

func TestRobotsTxtFetchMetricsCountEveryFetchByOutcome(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := robotstxtfetchobserversprometheus.New(registry)
	robotsTxtURL := canonicalurltest.CanonicalURLOf(t, "http://host.example/robots.txt")

	metrics.RobotsTxtAnswered(context.Background(), robotsTxtURL)
	metrics.RobotsTxtAnswered(context.Background(), robotsTxtURL)
	metrics.RobotsTxtUnreachable(context.Background(), robotsTxtURL, errors.New("refused"))

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+fetchesMetricName+" Fetches of robots.txt, by whether the site answered.\n"+
			"# TYPE "+fetchesMetricName+" counter\n"+
			fetchesMetricName+`{outcome="answered"} 2`+"\n"+
			fetchesMetricName+`{outcome="canceled"} 0`+"\n"+
			fetchesMetricName+`{outcome="unreachable"} 1`+"\n",
	), fetchesMetricName); err != nil {
		t.Fatal(err)
	}
}

func TestACanceledRobotsTxtFetchIsCounted(t *testing.T) {
	registry := prometheusclient.NewRegistry()
	metrics := robotstxtfetchobserversprometheus.New(registry)

	metrics.RobotsTxtFetchCanceled(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host.example/robots.txt"),
	)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(
		"# HELP "+fetchesMetricName+" Fetches of robots.txt, by whether the site answered.\n"+
			"# TYPE "+fetchesMetricName+" counter\n"+
			fetchesMetricName+`{outcome="answered"} 0`+"\n"+
			fetchesMetricName+`{outcome="canceled"} 1`+"\n"+
			fetchesMetricName+`{outcome="unreachable"} 0`+"\n",
	), fetchesMetricName); err != nil {
		t.Fatal(err)
	}
}
