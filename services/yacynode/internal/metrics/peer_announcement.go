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
			Help: "Peers this node last contacted, by the peer type they reported this " +
				"node to be.",
		},
		[]string{labelContactOutcome},
	)
	reachablePeersPerScheme := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "yacynode_peerannouncement_reachable_peers_per_scheme",
			Help: "Peers confirmed reachable when this node last contacted its peers, by " +
				"the scheme of the origin that answered the greet.",
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
