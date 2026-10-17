package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const peerMetadataOutcomesMetric = `
# HELP yacynode_urlmeta_peer_metadata_total URL metadata records sent by peers, by what the node decided for them.
# TYPE yacynode_urlmeta_peer_metadata_total counter
yacynode_urlmeta_peer_metadata_total{outcome="replaced_older_version"} 2
yacynode_urlmeta_peer_metadata_total{outcome="dropped_already_stored"} 1
`

func TestURLMetadataIntakeCountsTheRecordsOfEachOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewURLMetadataIntakeMetrics(registry)

	observer.ObserveOutcome(urlmeta.OutcomeReplacedOlderVersion, 1)
	observer.ObserveOutcome(urlmeta.OutcomeReplacedOlderVersion, 1)
	observer.ObserveOutcome(urlmeta.OutcomeDroppedAlreadyStored, 1)

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(peerMetadataOutcomesMetric),
		"yacynode_urlmeta_peer_metadata_total",
	); err != nil {
		t.Fatal(err)
	}
}
