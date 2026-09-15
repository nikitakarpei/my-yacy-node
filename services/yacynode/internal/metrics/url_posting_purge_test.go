package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestURLPostingPurgeCountsThePostingsPurgedWithTheirURLs(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLPostingPurgeMetrics(registry)

	observer.ObservePostingsPurgedWithURL(urlHash(t, "a"), 3)
	observer.ObservePostingsPurgedWithURL(urlHash(t, "b"), 4)

	expected := `
# HELP yacynode_url_posting_purge_postings_total Postings purged with the url metadata that referenced them.
# TYPE yacynode_url_posting_purge_postings_total counter
yacynode_url_posting_purge_postings_total 7
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_url_posting_purge_postings_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func urlHash(t *testing.T, seed string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.URLHashOf("http://example.com/" + seed)
	if err != nil {
		t.Fatalf("URLHashOf: %v", err)
	}

	return hash
}
