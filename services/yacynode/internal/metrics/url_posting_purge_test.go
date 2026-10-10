package metrics_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

const purgedPostingsMetric = `
# HELP yacynode_url_posting_purge_postings_total Postings purged with the url metadata that referenced them.
# TYPE yacynode_url_posting_purge_postings_total counter
yacynode_url_posting_purge_postings_total %d
`

func TestURLPostingPurgeCountsThePostingsPurgedWithTheirURLs(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLPostingPurgeMetrics(registry)

	if err := openVault(t).Update(context.Background(), func(tx *vault.Txn) error {
		observer.ObservePostingsPurgedWithURL(tx, urlHash(t, "a"), 3)
		observer.ObservePostingsPurgedWithURL(tx, urlHash(t, "b"), 4)

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	assertPurgedPostings(t, registry, 7)
}

func TestURLPostingPurgeLeavesOutThePostingsOfAnAbortedWrite(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLPostingPurgeMetrics(registry)
	aborted := errors.New("aborted")

	if err := openVault(t).Update(context.Background(), func(tx *vault.Txn) error {
		observer.ObservePostingsPurgedWithURL(tx, urlHash(t, "a"), 3)

		return aborted
	}); !errors.Is(err, aborted) {
		t.Fatalf("Update = %v, want %v", err, aborted)
	}

	assertPurgedPostings(t, registry, 0)
}

func assertPurgedPostings(t *testing.T, registry *prometheus.Registry, postings int) {
	t.Helper()

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(fmt.Sprintf(purgedPostingsMetric, postings)),
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
