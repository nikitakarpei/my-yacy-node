package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
)

const labelContactOutcome = "outcome"

type PeerAnnouncementMetrics struct {
	peersPerContactOutcome *prometheus.GaugeVec
}

func NewPeerAnnouncementMetrics(registry prometheus.Registerer) *PeerAnnouncementMetrics {
	peersPerContactOutcome := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "yacynode_peerannouncement_peers_per_contact_outcome",
			Help: "Peers contacted in the last announce round, by the peer type they " +
				"reported this node to be.",
		},
		[]string{labelContactOutcome},
	)
	registry.MustRegister(peersPerContactOutcome)

	return &PeerAnnouncementMetrics{peersPerContactOutcome: peersPerContactOutcome}
}

func (p *PeerAnnouncementMetrics) ObserveAnnounceRound(
	amountOfPeersPerContactOutcome map[peerannouncement.ContactOutcome]int,
) {
	for outcome, amountOfPeers := range amountOfPeersPerContactOutcome {
		p.peersPerContactOutcome.
			WithLabelValues(string(outcome)).
			Set(float64(amountOfPeers))
	}
}
