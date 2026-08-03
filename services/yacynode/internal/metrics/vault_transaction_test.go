package metrics

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestVaultTransactionReportsBeginLatency(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteBegan(time.Millisecond)

	if count := histogramCount(
		t,
		registry,
		"vault_write_transaction_begin_seconds",
		"",
		"",
	); count != 1 {
		t.Errorf("begin seconds count = %d, want 1", count)
	}
}

func TestVaultTransactionCountsBeginRefusalsByCause(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteBeginRefused("no_space")
	transactions.ObserveWriteBeginRefused("no_space")
	transactions.ObserveWriteBeginRefused("unclassified")

	expected := `
# HELP vault_write_transaction_begin_refusals_total Vault write transactions the storage engine refused before opening, by cause.
# TYPE vault_write_transaction_begin_refusals_total counter
vault_write_transaction_begin_refusals_total{cause="no_space"} 2
vault_write_transaction_begin_refusals_total{cause="unclassified"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"vault_write_transaction_begin_refusals_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultTransactionCountsOutcomes(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := NewVaultTransactionMetrics(registry)

	transactions.ObserveWriteCommitted(time.Millisecond, 2*time.Millisecond)
	transactions.ObserveWriteAborted(time.Millisecond, time.Millisecond)
	transactions.ObserveWriteCommitRefused(time.Millisecond, time.Millisecond, "no_space")

	expected := `
# HELP vault_write_transactions_total Vault write transactions that opened, by how they ended.
# TYPE vault_write_transactions_total counter
vault_write_transactions_total{cause="",outcome="aborted"} 1
vault_write_transactions_total{cause="",outcome="committed"} 1
vault_write_transactions_total{cause="no_space",outcome="refused"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"vault_write_transactions_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}

	if count := histogramCount(
		t,
		registry,
		"vault_write_transaction_duration_seconds",
		"phase",
		"commit",
	); count != 1 {
		t.Errorf("commit duration count = %d, want 1", count)
	}
	if count := histogramCount(
		t,
		registry,
		"vault_write_transaction_duration_seconds",
		"phase",
		"rollback",
	); count != 2 {
		t.Errorf("rollback duration count = %d, want 2", count)
	}
	if count := histogramCount(
		t,
		registry,
		"vault_write_transaction_duration_seconds",
		"phase",
		"execute",
	); count != 3 {
		t.Errorf("execute duration count = %d, want 3", count)
	}
}

func TestVaultTransactionTracksReadsInFlight(t *testing.T) {
	registry := prometheus.NewRegistry()
	transactions := NewVaultTransactionMetrics(registry)

	transactions.ObserveReadBegan()
	transactions.ObserveReadBegan()
	transactions.ObserveReadEnded()

	expected := `
# HELP vault_reads_in_flight Reads currently open through the vault.
# TYPE vault_reads_in_flight gauge
vault_reads_in_flight 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"vault_reads_in_flight",
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
