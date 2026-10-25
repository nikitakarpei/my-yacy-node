package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

const (
	labelContactOutcome = "outcome"
	labelScheme         = "scheme"
)

type PeerAnnouncementMetrics struct {
	peersPerContactOutcome  *prometheus.GaugeVec
	reachablePeersPerScheme *prometheus.GaugeVec
}

func NewPeerAnnouncementMetrics(registry prometheus.Registerer) *PeerAnnouncementMetrics {
	peersPerContactOutcome := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "yacynode_peerannouncement_peers_per_contact_outcome",
			Help: "Peers greeted at the last announce, by the peer type they reported " +
				"this node to be.",
		},
		[]string{labelContactOutcome},
	)
	reachablePeersPerScheme := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "yacynode_peerannouncement_reachable_peers_per_scheme",
			Help: "Peers that answered the last announce, by the scheme of the base URL " +
				"that answered.",
		},
		[]string{labelScheme},
	)
	registry.MustRegister(peersPerContactOutcome, reachablePeersPerScheme)

	return &PeerAnnouncementMetrics{
		peersPerContactOutcome:  peersPerContactOutcome,
		reachablePeersPerScheme: reachablePeersPerScheme,
	}
}

func (p *PeerAnnouncementMetrics) ObservePeerContacts(
	amountOfPeersPerContactOutcome map[peerannouncement.ContactOutcome]int,
	amountOfReachablePeersPerScheme map[peerwire.Scheme]int,
) {
	for outcome, amountOfPeers := range amountOfPeersPerContactOutcome {
		p.peersPerContactOutcome.
			WithLabelValues(string(outcome)).
			Set(float64(amountOfPeers))
	}
	for scheme, amountOfPeers := range amountOfReachablePeersPerScheme {
		p.reachablePeersPerScheme.
			WithLabelValues(string(scheme)).
			Set(float64(amountOfPeers))
	}
}
