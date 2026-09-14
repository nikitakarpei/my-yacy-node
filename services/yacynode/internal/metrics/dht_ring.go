package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	labelDHTRingSector  = "sector"
	dhtRingSectorFormat = "%02d"
)

type DHTRingMetrics struct {
	peersAcceptingRemoteIndexPerSector *prometheus.GaugeVec
	ringFractionFromPostingToHolder    prometheus.Histogram
}

func NewDHTRingMetrics(registry prometheus.Registerer) *DHTRingMetrics {
	peersAcceptingRemoteIndexPerSector := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "yacynode_rwidistribution_peers_accepting_remote_index",
			Help: "Peers accepting remote index, by the DHT ring sector they occupy.",
		},
		[]string{labelDHTRingSector},
	)
	ringFractionFromPostingToHolder := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "yacynode_rwidistribution_replica_ring_fraction",
		Help: "Fraction of the DHT ring between a posting's position and the peer " +
			"that accepted its replica.",
		Buckets: prometheus.ExponentialBucketsRange(1e-6, 1, 13),
	})
	registry.MustRegister(peersAcceptingRemoteIndexPerSector, ringFractionFromPostingToHolder)

	return &DHTRingMetrics{
		peersAcceptingRemoteIndexPerSector: peersAcceptingRemoteIndexPerSector,
		ringFractionFromPostingToHolder:    ringFractionFromPostingToHolder,
	}
}

func (d *DHTRingMetrics) ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int) {
	for sector, peers := range peersPerSector {
		d.peersAcceptingRemoteIndexPerSector.
			WithLabelValues(fmt.Sprintf(dhtRingSectorFormat, sector)).
			Set(float64(peers))
	}
}

func (d *DHTRingMetrics) ObserveReplicaRingFractions(ringFractions []float64) {
	for _, fraction := range ringFractions {
		d.ringFractionFromPostingToHolder.Observe(fraction)
	}
}
