package metrics

import (
	"context"
	"net/netip"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/asmap"
)

const labelASN = "asn"

type ReachablePeers interface {
	ReachablePeers(ctx context.Context) []yacymodel.Seed
}

type AutonomousSystems interface {
	ASNOf(address netip.Addr) asmap.ASN
}

type ReachablePeerASNMetrics struct {
	roster               ReachablePeers
	autonomousSystems    AutonomousSystems
	reachablePeersPerASN *prometheus.Desc
}

func NewReachablePeerASNMetrics(
	registry prometheus.Registerer,
	roster ReachablePeers,
	autonomousSystems AutonomousSystems,
) *ReachablePeerASNMetrics {
	metrics := &ReachablePeerASNMetrics{
		roster:            roster,
		autonomousSystems: autonomousSystems,
		reachablePeersPerASN: prometheus.NewDesc(
			"yacynode_peerroster_reachable_peers_per_asn",
			"Peers currently confirmed reachable, by the autonomous system of their IP "+
				"addresses. A peer counts once in each autonomous system of its addresses.",
			[]string{labelASN},
			nil,
		),
	}
	registry.MustRegister(metrics)

	return metrics
}

func (m *ReachablePeerASNMetrics) Describe(descriptions chan<- *prometheus.Desc) {
	descriptions <- m.reachablePeersPerASN
}

func (m *ReachablePeerASNMetrics) Collect(samples chan<- prometheus.Metric) {
	for asn, amountOfPeers := range m.amountOfPeersPerASN(context.Background()) {
		samples <- prometheus.MustNewConstMetric(
			m.reachablePeersPerASN,
			prometheus.GaugeValue,
			float64(amountOfPeers),
			strconv.FormatUint(uint64(asn), 10),
		)
	}
}

func (m *ReachablePeerASNMetrics) amountOfPeersPerASN(ctx context.Context) map[asmap.ASN]int {
	amountPerASN := make(map[asmap.ASN]int)
	for _, peer := range m.roster.ReachablePeers(ctx) {
		for asn := range m.asnsOf(peer) {
			amountPerASN[asn]++
		}
	}

	return amountPerASN
}

func (m *ReachablePeerASNMetrics) asnsOf(peer yacymodel.Seed) map[asmap.ASN]struct{} {
	addresses, _ := peer.Addresses()
	asns := make(map[asmap.ASN]struct{}, len(addresses.Hosts))
	for _, host := range addresses.Hosts {
		if address, isAddress := host.Addr(); isAddress {
			asns[m.autonomousSystems.ASNOf(address)] = struct{}{}
		}
	}

	return asns
}
