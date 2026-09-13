package metrics_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

type stubRingOccupancy struct {
	occupancyPerSector map[yacymodel.DHTRingSector]int
	err                error
}

func (s stubRingOccupancy) OccupancyOfDHTRingSectors(
	*vault.Txn,
) (map[yacymodel.DHTRingSector]int, error) {
	return s.occupancyPerSector, s.err
}

func openVault(t *testing.T) *vault.Vault {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return v
}

func TestRWIRingOccupancyReportsOccupiedSectorsAndTheSectorOfTheNode(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewRWIRingOccupancyMetrics(
		registry,
		openVault(t),
		stubRingOccupancy{occupancyPerSector: map[yacymodel.DHTRingSector]int{3: 5, 41: 2}},
		yacymodel.DHTRingSector(3),
	)

	expected := `
# HELP yacynode_dht_ring_sector_of_node DHT ring sector this node's own position falls in.
# TYPE yacynode_dht_ring_sector_of_node gauge
yacynode_dht_ring_sector_of_node 3
# HELP yacynode_rwi_ring_sector_occupancy Occupancy of a DHT ring sector by the postings this node holds.
# TYPE yacynode_rwi_ring_sector_occupancy gauge
yacynode_rwi_ring_sector_occupancy{sector="03"} 5
yacynode_rwi_ring_sector_occupancy{sector="41"} 2
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestRWIRingOccupancyOmitsOccupiedSectorsOnError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewRWIRingOccupancyMetrics(
		registry,
		openVault(t),
		stubRingOccupancy{err: errors.New("unavailable")},
		yacymodel.DHTRingSector(3),
	)

	if got := testutil.CollectAndCount(
		registry,
		"yacynode_rwi_ring_sector_occupancy",
	); got != 0 {
		t.Errorf("ring occupancy samples = %d, want none on error", got)
	}
	if got := testutil.CollectAndCount(registry, "yacynode_dht_ring_sector_of_node"); got != 1 {
		t.Errorf("sector of node samples = %d, want 1", got)
	}
}
