package metrics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RWIPostingSectorAmounts interface {
	AmountOfPostingsPerDHTRingSector(
		tx *vault.Txn,
	) (map[yacymodel.DHTRingSector]int, error)
}

type RWIPostingSectorMetrics struct {
	vault      *vault.Vault
	amounts    RWIPostingSectorAmounts
	nodeSector yacymodel.DHTRingSector
	postings   *prometheus.Desc
	ofNode     *prometheus.Desc
}

func NewRWIPostingSectorMetrics(
	registry prometheus.Registerer,
	v *vault.Vault,
	amounts RWIPostingSectorAmounts,
	nodeSector yacymodel.DHTRingSector,
) *RWIPostingSectorMetrics {
	metrics := &RWIPostingSectorMetrics{
		vault:      v,
		amounts:    amounts,
		nodeSector: nodeSector,
		postings: prometheus.NewDesc(
			"yacynode_rwipostings_per_dht_ring_sector",
			"Postings this node holds, by the DHT ring sector they fall in.",
			[]string{labelDHTRingSector},
			nil,
		),
		ofNode: prometheus.NewDesc(
			"yacynode_dht_ring_sector_of_node",
			"DHT ring sector this node's own position falls in.",
			nil,
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *RWIPostingSectorMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- m.postings
	descriptions <- m.ofNode
}

func (m *RWIPostingSectorMetrics) Collect(samples chan<- prometheus.Metric) {
	ctx := context.Background()

	samples <- prometheus.MustNewConstMetric(
		m.ofNode,
		prometheus.GaugeValue,
		float64(m.nodeSector),
	)

	amountPerSector, err := m.amountOfPostingsPerDHTRingSector(ctx)
	if err != nil {
		slog.WarnContext(ctx, "postings per dht ring sector unavailable", slog.Any("error", err))

		return
	}

	for sector, amountOfPostings := range amountPerSector {
		samples <- prometheus.MustNewConstMetric(
			m.postings,
			prometheus.GaugeValue,
			float64(amountOfPostings),
			fmt.Sprintf(dhtRingSectorFormat, sector),
		)
	}
}

func (m *RWIPostingSectorMetrics) amountOfPostingsPerDHTRingSector(
	ctx context.Context,
) (map[yacymodel.DHTRingSector]int, error) {
	var amountPerSector map[yacymodel.DHTRingSector]int
	if err := m.vault.View(ctx, func(tx *vault.Txn) error {
		amounts, err := m.amounts.AmountOfPostingsPerDHTRingSector(tx)
		amountPerSector = amounts

		return err
	}); err != nil {
		return nil, fmt.Errorf("read posting amounts per sector: %w", err)
	}

	return amountPerSector, nil
}
