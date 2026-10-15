package metrics_test

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

const pageVersionVerdictsMetric = `
# HELP yacynode_urlpageversion_postings_total Peer postings by what the node did with them and why, against the page version it indexed.
# TYPE yacynode_urlpageversion_postings_total counter
yacynode_urlpageversion_postings_total{outcome="dropped",reason="older_version"} 1
yacynode_urlpageversion_postings_total{outcome="indexed",reason="newer_version"} 2
`

func TestURLPageVersionCountsEachVerdictOfACommittedWrite(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLPageVersionMetrics(registry)

	if err := openVault(t).Update(context.Background(), func(tx *vault.Txn) error {
		observer.ObserveVerdict(tx, urlpageversion.VerdictNewerVersion)
		observer.ObserveVerdict(tx, urlpageversion.VerdictNewerVersion)
		observer.ObserveVerdict(tx, urlpageversion.VerdictOlderVersion)

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(pageVersionVerdictsMetric),
		"yacynode_urlpageversion_postings_total",
	); err != nil {
		t.Fatal(err)
	}
}
