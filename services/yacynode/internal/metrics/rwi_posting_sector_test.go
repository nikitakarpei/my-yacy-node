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

type stubPostingSectorAmounts struct {
	amountPerSector map[yacymodel.DHTRingSector]int
	err             error
}

func (s stubPostingSectorAmounts) AmountOfPostingsPerDHTRingSector(
	*vault.Txn,
) (map[yacymodel.DHTRingSector]int, error) {
	return s.amountPerSector, s.err
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

func TestRWIPostingSectorReportsPostingsPerSectorAndTheSectorOfTheNode(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewRWIPostingSectorMetrics(
		registry,
		openVault(t),
		stubPostingSectorAmounts{amountPerSector: map[yacymodel.DHTRingSector]int{3: 5, 41: 2}},
		yacymodel.DHTRingSector(3),
	)

	expected := `
# HELP yacynode_dht_ring_sector_of_node DHT ring sector this node's own position falls in.
# TYPE yacynode_dht_ring_sector_of_node gauge
yacynode_dht_ring_sector_of_node 3
# HELP yacynode_rwipostings_per_dht_ring_sector Postings this node holds, by the DHT ring sector they fall in.
# TYPE yacynode_rwipostings_per_dht_ring_sector gauge
yacynode_rwipostings_per_dht_ring_sector{sector="03"} 5
yacynode_rwipostings_per_dht_ring_sector{sector="41"} 2
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestRWIPostingSectorOmitsPostingsPerSectorOnError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewRWIPostingSectorMetrics(
		registry,
		openVault(t),
		stubPostingSectorAmounts{err: errors.New("unavailable")},
		yacymodel.DHTRingSector(3),
	)

	if got := testutil.CollectAndCount(
		registry,
		"yacynode_rwipostings_per_dht_ring_sector",
	); got != 0 {
		t.Errorf("postings per sector samples = %d, want none on error", got)
	}
	if got := testutil.CollectAndCount(registry, "yacynode_dht_ring_sector_of_node"); got != 1 {
		t.Errorf("sector of node samples = %d, want 1", got)
	}
}
