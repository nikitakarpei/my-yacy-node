// Package prometheus reports peer directory changes and how full it is.
package prometheus

import (
	"context"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	labelChange          = "change"
	changePeerAdmitted   = "admitted"
	changePeerAnswered   = "answered"
	changePeerWentSilent = "wentSilent"
	changePeerDropped    = "dropped"
)

type DirectoryMetrics struct {
	peersAdmitted           prometheusclient.Counter
	peersAnswered           prometheusclient.Counter
	peersWentSilent         prometheusclient.Counter
	peersDropped            prometheusclient.Counter
	directoryPeers          prometheusclient.Gauge
	directoryAnsweringPeers prometheusclient.Gauge
	directoryCapacity       prometheusclient.Gauge
}

func New(registry prometheusclient.Registerer) *DirectoryMetrics {
	peerChanges := prometheusclient.NewCounterVec(prometheusclient.CounterOpts{
		Name: "yacydhtsearch_directory_peer_changes_total",
		Help: "Peer directory changes, by change.",
	}, []string{labelChange})
	metrics := &DirectoryMetrics{
		peersAdmitted:   peerChanges.WithLabelValues(changePeerAdmitted),
		peersAnswered:   peerChanges.WithLabelValues(changePeerAnswered),
		peersWentSilent: peerChanges.WithLabelValues(changePeerWentSilent),
		peersDropped:    peerChanges.WithLabelValues(changePeerDropped),
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
		peerChanges,
		metrics.directoryPeers,
		metrics.directoryAnsweringPeers,
		metrics.directoryCapacity,
	)

	return metrics
}

func (m *DirectoryMetrics) PeerAdmitted(context.Context, yacymodel.Hash, int) {
	m.peersAdmitted.Inc()
}

func (m *DirectoryMetrics) PeerAnswered(context.Context, yacymodel.Hash, string, time.Time) {
	m.peersAnswered.Inc()
}

func (m *DirectoryMetrics) PeerWentSilent(context.Context, yacymodel.Hash) {
	m.peersWentSilent.Inc()
}

func (m *DirectoryMetrics) PeerDropped(context.Context, yacymodel.Hash) {
	m.peersDropped.Inc()
}

func (m *DirectoryMetrics) PeersKnown(
	_ context.Context,
	peers, answeringPeers, capacity int,
) {
	m.directoryPeers.Set(float64(peers))
	m.directoryAnsweringPeers.Set(float64(answeringPeers))
	m.directoryCapacity.Set(float64(capacity))
}
