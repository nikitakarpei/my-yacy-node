package metrics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RWIRingOccupancy interface {
	OccupancyOfDHTRingSectors(
		tx *vault.Txn,
	) (map[yacymodel.DHTRingSector]int, error)
}

type RWIRingOccupancyMetrics struct {
	vault           *vault.Vault
	occupancy       RWIRingOccupancy
	nodeSector      yacymodel.DHTRingSector
	sectorOccupancy *prometheus.Desc
	sectorOfNode    *prometheus.Desc
}

func NewRWIRingOccupancyMetrics(
	registry prometheus.Registerer,
	v *vault.Vault,
	occupancy RWIRingOccupancy,
	nodeSector yacymodel.DHTRingSector,
) *RWIRingOccupancyMetrics {
	metrics := &RWIRingOccupancyMetrics{
		vault:      v,
		occupancy:  occupancy,
		nodeSector: nodeSector,
		sectorOccupancy: prometheus.NewDesc(
			"yacynode_rwi_ring_sector_occupancy",
			"Occupancy of a DHT ring sector by the postings this node holds.",
			[]string{labelDHTRingSector},
			nil,
		),
		sectorOfNode: prometheus.NewDesc(
			"yacynode_dht_ring_sector_of_node",
			"DHT ring sector this node's own position falls in.",
			nil,
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *RWIRingOccupancyMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- m.sectorOccupancy
	descriptions <- m.sectorOfNode
}

func (m *RWIRingOccupancyMetrics) Collect(samples chan<- prometheus.Metric) {
	ctx := context.Background()

	samples <- prometheus.MustNewConstMetric(
		m.sectorOfNode,
		prometheus.GaugeValue,
		float64(m.nodeSector),
	)

	occupancyPerSector, err := m.occupancyOfDHTRingSectors(ctx)
	if err != nil {
		slog.WarnContext(ctx, "ring occupancy of sectors unavailable", slog.Any("error", err))

		return
	}

	for sector, amountOfPostings := range occupancyPerSector {
		samples <- prometheus.MustNewConstMetric(
			m.sectorOccupancy,
			prometheus.GaugeValue,
			float64(amountOfPostings),
			fmt.Sprintf(dhtRingSectorFormat, sector),
		)
	}
}

func (m *RWIRingOccupancyMetrics) occupancyOfDHTRingSectors(
	ctx context.Context,
) (map[yacymodel.DHTRingSector]int, error) {
	var occupancyPerSector map[yacymodel.DHTRingSector]int
	if err := m.vault.View(ctx, func(tx *vault.Txn) error {
		occupancy, err := m.occupancy.OccupancyOfDHTRingSectors(tx)
		occupancyPerSector = occupancy

		return err
	}); err != nil {
		return nil, fmt.Errorf("read ring occupancy of sectors: %w", err)
	}

	return occupancyPerSector, nil
}
