package metrics_test

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
)

const admissionOutcomesMetric = `
# HELP yacynode_rwiadmission_postings_total Inbound postings by what happened to them.
# TYPE yacynode_rwiadmission_postings_total counter
yacynode_rwiadmission_postings_total{outcome="indexed_newer_version"} 2
yacynode_rwiadmission_postings_total{outcome="dropped_same_version"} 1
`

func TestRWIAdmissionCountsEachOutcomeOfACommittedWrite(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewRWIAdmissionMetrics(registry)

	if err := openVault(t).Update(context.Background(), func(tx *vault.Txn) error {
		observer.ObserveOutcome(tx, rwiadmission.OutcomeIndexedNewerVersion)
		observer.ObserveOutcome(tx, rwiadmission.OutcomeIndexedNewerVersion)
		observer.ObserveOutcome(tx, rwiadmission.OutcomeDroppedSameVersion)

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(admissionOutcomesMetric),
		"yacynode_rwiadmission_postings_total",
	); err != nil {
		t.Fatal(err)
	}
}
