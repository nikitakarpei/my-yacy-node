package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PeerRosterMetrics struct {
	knownPeers     prometheus.Gauge
	reachablePeers prometheus.Gauge
}

func NewPeerRosterMetrics(registry prometheus.Registerer) *PeerRosterMetrics {
	knownPeers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "peerroster_known_peers",
		Help: "Peers currently known to this node's roster.",
	})
	reachablePeers := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "peerroster_reachable_peers",
		Help: "Peers currently confirmed reachable.",
	})
	registry.MustRegister(knownPeers, reachablePeers)

	return &PeerRosterMetrics{knownPeers: knownPeers, reachablePeers: reachablePeers}
}

func (p *PeerRosterMetrics) ObserveKnownPeers(count int) {
	p.knownPeers.Set(float64(count))
}

func (p *PeerRosterMetrics) ObserveReachablePeers(count int) {
	p.reachablePeers.Set(float64(count))
}
