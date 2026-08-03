package metrics

import (
	"context"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const labelCollection = "collection"

type VaultCollections interface {
	EntriesByCollection(context.Context) (map[vault.Name]int, error)
}

type VaultCollectionMetrics struct {
	collections VaultCollections
	entries     *prometheus.Desc
}

func NewVaultCollectionMetrics(
	registry prometheus.Registerer,
	collections VaultCollections,
) *VaultCollectionMetrics {
	metrics := &VaultCollectionMetrics{
		collections: collections,
		entries: prometheus.NewDesc(
			"vault_collection_entries",
			"Entries currently stored in each vault collection.",
			[]string{labelCollection},
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *VaultCollectionMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- m.entries
}

func (m *VaultCollectionMetrics) Collect(samples chan<- prometheus.Metric) {
	ctx := context.Background()

	entries, err := m.collections.EntriesByCollection(ctx)
	if err != nil {
		slog.WarnContext(ctx, "vault collection entries unavailable", slog.Any("error", err))

		return
	}

	for collection, count := range entries {
		samples <- prometheus.MustNewConstMetric(
			m.entries,
			prometheus.GaugeValue,
			float64(count),
			string(collection),
		)
	}
}
