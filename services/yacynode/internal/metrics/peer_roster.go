package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

type PeerRosterMetrics struct {
	knownPeers     prometheus.Gauge
	reachablePeers prometheus.Gauge
	droppedSeeds   *prometheus.CounterVec
}

func NewPeerRosterMetrics(registry prometheus.Registerer) *PeerRosterMetrics {
	knownPeers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "yacynode_peerroster_known_peers",
		Help: "Peers currently known to this node's roster.",
	})
	reachablePeers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "yacynode_peerroster_reachable_peers",
		Help: "Peers currently confirmed reachable.",
	})
	droppedSeeds := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "yacynode_peerroster_dropped_seeds_total",
		Help: "Peer seeds dropped because they break the identity bound to the peer, by reason.",
	}, []string{"reason"})
	registry.MustRegister(knownPeers, reachablePeers, droppedSeeds)

	return &PeerRosterMetrics{
		knownPeers:     knownPeers,
		reachablePeers: reachablePeers,
		droppedSeeds:   droppedSeeds,
	}
}

func (p *PeerRosterMetrics) ObserveKnownPeers(count int) {
	p.knownPeers.Set(float64(count))
}

func (p *PeerRosterMetrics) ObserveReachablePeers(count int) {
	p.reachablePeers.Set(float64(count))
}

func (p *PeerRosterMetrics) ObserveDroppedSeed(reason peerroster.SeedDropReason) {
	p.droppedSeeds.WithLabelValues(string(reason)).Inc()
}
