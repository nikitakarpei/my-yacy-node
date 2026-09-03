package metrics_test

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestVaultTransactionReportsBeginLatency(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := metrics.NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteBegan(time.Millisecond)

	if count := histogramCount(
		t,
		registry,
		"yacynode_vault_write_transaction_begin_seconds",
		"",
		"",
	); count != 1 {
		t.Errorf("begin seconds count = %d, want 1", count)
	}
}

func TestVaultTransactionCountsBeginRefusalsByCause(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := metrics.NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteBeginRefused("no_space")
	transactions.ObserveWriteBeginRefused("no_space")
	transactions.ObserveWriteBeginRefused("unclassified")

	expected := `
# HELP yacynode_vault_write_transaction_begin_refusals_total Vault write transactions the storage engine refused before opening, by cause.
# TYPE yacynode_vault_write_transaction_begin_refusals_total counter
yacynode_vault_write_transaction_begin_refusals_total{cause="no_space"} 2
yacynode_vault_write_transaction_begin_refusals_total{cause="unclassified"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_vault_write_transaction_begin_refusals_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultTransactionCountsOutcomes(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := metrics.NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteCommitted(time.Millisecond, 2*time.Millisecond, true)
	transactions.ObserveWriteAborted(time.Millisecond, time.Millisecond)
	transactions.ObserveWriteCommitRefused(time.Millisecond, time.Millisecond, "no_space")

	expected := `
# HELP yacynode_vault_write_transactions_total Vault write transactions that opened, by how they ended.
# TYPE yacynode_vault_write_transactions_total counter
yacynode_vault_write_transactions_total{called_write_operation="",cause="",outcome="aborted"} 1
yacynode_vault_write_transactions_total{called_write_operation="",cause="no_space",outcome="refused"} 1
yacynode_vault_write_transactions_total{called_write_operation="true",cause="",outcome="committed"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_vault_write_transactions_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}

	if count := histogramCount(
		t,
		registry,
		"yacynode_vault_write_transaction_duration_seconds",
		"phase",
		"commit",
	); count != 1 {
		t.Errorf("commit duration count = %d, want 1", count)
	}
	if count := histogramCount(
		t,
		registry,
		"yacynode_vault_write_transaction_duration_seconds",
		"phase",
		"rollback",
	); count != 2 {
		t.Errorf("rollback duration count = %d, want 2", count)
	}
	if count := histogramCount(
		t,
		registry,
		"yacynode_vault_write_transaction_duration_seconds",
		"phase",
		"execute",
	); count != 3 {
		t.Errorf("execute duration count = %d, want 3", count)
	}
}

func TestVaultTransactionSeparatesCommitsThatCalledNoWriteOperation(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := metrics.NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteCommitted(time.Millisecond, time.Millisecond, true)
	transactions.ObserveWriteCommitted(time.Millisecond, time.Millisecond, false)
	transactions.ObserveWriteCommitted(time.Millisecond, time.Millisecond, false)

	expected := `
# HELP yacynode_vault_write_transactions_total Vault write transactions that opened, by how they ended.
# TYPE yacynode_vault_write_transactions_total counter
yacynode_vault_write_transactions_total{called_write_operation="false",cause="",outcome="committed"} 2
yacynode_vault_write_transactions_total{called_write_operation="true",cause="",outcome="committed"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_vault_write_transactions_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultTransactionTracksReadsInFlight(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := metrics.NewVaultTransactionMetrics(registry)

	transactions.ObserveReadBegan()
	transactions.ObserveReadBegan()
	transactions.ObserveReadEnded()

	expected := `
# HELP yacynode_vault_reads_in_flight Reads currently open through the vault.
# TYPE yacynode_vault_reads_in_flight gauge
yacynode_vault_reads_in_flight 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_vault_reads_in_flight",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func histogramCount(
	t *testing.T,
	gatherer prometheus.Gatherer,
	family string,
	labelName string,
	labelValue string,
) uint64 {
	t.Helper()

	families, err := gatherer.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}

	for _, mf := range families {
		if mf.GetName() != family {
			continue
		}
		for _, metric := range mf.GetMetric() {
			if labelName == "" {
				return metric.GetHistogram().GetSampleCount()
			}
			for _, label := range metric.GetLabel() {
				if label.GetName() == labelName && label.GetValue() == labelValue {
					return metric.GetHistogram().GetSampleCount()
				}
			}
		}
	}

	return 0
}
