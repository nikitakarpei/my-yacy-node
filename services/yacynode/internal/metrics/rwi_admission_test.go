package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
)

const admissionOutcomesMetric = `
# HELP yacynode_rwiadmission_postings_total Inbound postings by what the node decided for them.
# TYPE yacynode_rwiadmission_postings_total counter
yacynode_rwiadmission_postings_total{outcome="indexed_newer_version"} 2
yacynode_rwiadmission_postings_total{outcome="refused_storage_full"} 3
`

func TestRWIAdmissionCountsThePostingsOfEachOutcome(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewRWIAdmissionMetrics(registry)

	observer.ObserveOutcome(rwiadmission.OutcomeIndexedNewerVersion, 1)
	observer.ObserveOutcome(rwiadmission.OutcomeIndexedNewerVersion, 1)
	observer.ObserveOutcome(rwiadmission.OutcomeRefusedStorageFull, 3)

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(admissionOutcomesMetric),
		"yacynode_rwiadmission_postings_total",
	); err != nil {
		t.Fatal(err)
	}
}
