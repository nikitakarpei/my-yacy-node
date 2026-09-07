// Package prometheus reports peer directory changes and how full it is.
package prometheus

import (
	"context"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	labelChange         = "change"
	changePeerAdmitted  = "admitted"
	changePeerAnswering = "answering"
	changePeerSilent    = "silent"
	changePeerDropped   = "dropped"
)

type DirectoryMetrics struct {
	peerChanges             *prometheusclient.CounterVec
	directoryPeers          prometheusclient.Gauge
	directoryAnsweringPeers prometheusclient.Gauge
	directoryCapacity       prometheusclient.Gauge
}

func New(registry prometheusclient.Registerer) *DirectoryMetrics {
	metrics := &DirectoryMetrics{
		peerChanges: prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
			Name: "yacydhtsearch_directory_peer_changes_total",
			Help: "Peer directory changes, by change.",
		}, []string{labelChange}),
		directoryPeers: prometheusclient.NewGauge(prometheusclient.GaugeOpts{
			Name: "yacydhtsearch_directory_peers",
			Help: "Peers the directory holds.",
		}),
		directoryAnsweringPeers: prometheusclient.NewGauge(prometheusclient.GaugeOpts{
			Name: "yacydhtsearch_directory_answering_peers",
			Help: "Peers the directory holds that answer on an address.",
		}),
		directoryCapacity: prometheusclient.NewGauge(prometheusclient.GaugeOpts{
			Name: "yacydhtsearch_directory_capacity",
			Help: "Most peers the directory may hold.",
		}),
	}
	registry.MustRegister(
		metrics.peerChanges,
		metrics.directoryPeers,
		metrics.directoryAnsweringPeers,
		metrics.directoryCapacity,
	)

	return metrics
}

func (m *DirectoryMetrics) PeerAdmitted(context.Context, yacymodel.Hash, int) {
	m.peerChanges.WithLabelValues(changePeerAdmitted).Inc()
}

func (m *DirectoryMetrics) PeerAnswering(context.Context, yacymodel.Hash, string) {
	m.peerChanges.WithLabelValues(changePeerAnswering).Inc()
}

func (m *DirectoryMetrics) PeerSilent(context.Context, yacymodel.Hash) {
	m.peerChanges.WithLabelValues(changePeerSilent).Inc()
}

func (m *DirectoryMetrics) PeerDropped(context.Context, yacymodel.Hash) {
	m.peerChanges.WithLabelValues(changePeerDropped).Inc()
}

func (m *DirectoryMetrics) DirectoryHolds(
	_ context.Context,
	peers, answeringPeers, capacity int,
) {
	m.directoryPeers.Set(float64(peers))
	m.directoryAnsweringPeers.Set(float64(answeringPeers))
	m.directoryCapacity.Set(float64(capacity))
}
