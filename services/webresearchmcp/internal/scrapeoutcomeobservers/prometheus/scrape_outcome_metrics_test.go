package prometheus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	scrapeoutcomemetricsprometheus "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/scrapeoutcomeobservers/prometheus"
)

const metricName = "webresearchmcp_scrape_outcome_listening_failures_total"

func TestScrapeOutcomeMetricsCountEveryListeningFailure(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := scrapeoutcomemetricsprometheus.New(registry)
	pageURL := canonicalurl.CanonicalURL{}

	metrics.ScrapeOutcomeSubscriptionFailed(
		context.Background(), pageURL, errors.New("no subscription"),
	)
	metrics.ScrapeOutcomeListenerConfirmationFailed(
		context.Background(), pageURL, errors.New("not confirmed"),
	)
	metrics.ScrapeOutcomeMessageMalformed(
		context.Background(), pageURL, errors.New("unreadable message"),
	)

	expected := `
# HELP webresearchmcp_scrape_outcome_listening_failures_total ` +
		`Failures met while listening for the end of one scrape, by outcome.
# TYPE webresearchmcp_scrape_outcome_listening_failures_total counter
webresearchmcp_scrape_outcome_listening_failures_total{outcome="listener_confirmation_failed"} 1
webresearchmcp_scrape_outcome_listening_failures_total{outcome="message_malformed"} 1
webresearchmcp_scrape_outcome_listening_failures_total{outcome="subscription_failed"} 1
`
	if err := testutil.GatherAndCompare(
		registry, strings.NewReader(expected), metricName,
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestEveryListeningFailureReadsZeroBeforeItHappens(t *testing.T) {
	registry := prometheus.NewRegistry()
	scrapeoutcomemetricsprometheus.New(registry)

	if got := testutil.CollectAndCount(registry, metricName); got != 3 {
		t.Fatalf("listening failure series = %d, want 3", got)
	}
}
