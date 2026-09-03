package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

func TestDHTRingTracksAcceptingPeersPerSector(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDHTRingMetrics(registry)

	observer.ObservePeersAcceptingRemoteIndexPerDHTRingSector([]int{0, 2, 0})

	expected := `
# HELP yacynode_rwidistribution_peers_accepting_remote_index Peers accepting remote index, by the DHT ring sector they occupy.
# TYPE yacynode_rwidistribution_peers_accepting_remote_index gauge
yacynode_rwidistribution_peers_accepting_remote_index{sector="00"} 0
yacynode_rwidistribution_peers_accepting_remote_index{sector="01"} 2
yacynode_rwidistribution_peers_accepting_remote_index{sector="02"} 0
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_rwidistribution_peers_accepting_remote_index",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestDHTRingTracksReplicaRingFractions(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewDHTRingMetrics(registry)

	observer.ObserveReplicaRingFractions([]float64{0.1, 0.4})

	if got := testutil.CollectAndCount(
		registry,
		"yacynode_rwidistribution_replica_ring_fraction",
	); got != 1 {
		t.Errorf("series = %v, want 1", got)
	}

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "yacynode_rwidistribution_replica_ring_fraction" {
			continue
		}
		histogram := family.GetMetric()[0].GetHistogram()
		if got := histogram.GetSampleCount(); got != 2 {
			t.Errorf("samples = %v, want one per reported fraction", got)
		}
		if got := histogram.GetSampleSum(); got < 0.49 || got > 0.51 {
			t.Errorf("sample sum = %v, want about 0.5", got)
		}
	}
}
