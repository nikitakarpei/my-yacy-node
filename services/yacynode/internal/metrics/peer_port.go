package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerport"
)

type PeerPortMetrics struct {
	connections *prometheus.CounterVec
}

func NewPeerPortMetrics(registry prometheus.Registerer) *PeerPortMetrics {
	connections := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "yacynode_peerport_connections_total",
		Help: "Connections to the peer port, by their scheme.",
	}, []string{"scheme"})
	for _, scheme := range []peerport.Scheme{peerport.HTTPScheme, peerport.HTTPSScheme} {
		connections.WithLabelValues(string(scheme))
	}
	registry.MustRegister(connections)

	return &PeerPortMetrics{connections: connections}
}

func (m *PeerPortMetrics) ObserveConnection(scheme peerport.Scheme) {
	m.connections.WithLabelValues(string(scheme)).Inc()
}
