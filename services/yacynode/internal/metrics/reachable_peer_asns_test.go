package metrics_test

import (
	"context"
	"net/netip"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/asmap"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
)

type fixedReachablePeers []yacymodel.Seed

func (p fixedReachablePeers) ReachablePeers(context.Context) []yacymodel.Seed {
	return p
}

type fixedAutonomousSystems map[string]asmap.ASN

func (a fixedAutonomousSystems) ASNOf(address netip.Addr) asmap.ASN {
	return a[address.String()]
}

func reachablePeerOn(t *testing.T, hosts ...string) yacymodel.Seed {
	t.Helper()
	parsedHosts := make([]yacymodel.Host, 0, len(hosts))
	for _, host := range hosts {
		parsedHost, err := yacymodel.ParseHost(host)
		if err != nil {
			t.Fatalf("ParseHost(%q): %v", host, err)
		}
		parsedHosts = append(parsedHosts, parsedHost)
	}

	return yacymodel.Seed{
		PrimaryHost:     yacymodel.Some(parsedHosts[0]),
		AdditionalHosts: yacymodel.Some(parsedHosts[1:]),
		Port:            yacymodel.Some(yacymodel.Port(8090)),
	}
}

func TestReachablePeerASNsCountEachPeerOncePerAutonomousSystem(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics.NewReachablePeerASNMetrics(
		registry,
		fixedReachablePeers{
			reachablePeerOn(t, "192.0.2.1"),
			reachablePeerOn(t, "192.0.2.2", "2001:db8::2"),
			reachablePeerOn(t, "198.51.100.3", "2001:db8::3"),
			reachablePeerOn(t, "peer.example.org"),
		},
		fixedAutonomousSystems{
			"192.0.2.1":    64500,
			"192.0.2.2":    64500,
			"2001:db8::2":  64500,
			"198.51.100.3": 64501,
			"2001:db8::3":  64502,
		},
	)

	expected := `
# HELP yacynode_peerroster_reachable_peers_per_asn Peers currently confirmed reachable, by the autonomous system of their IP addresses. A peer counts once in each autonomous system of its addresses.
# TYPE yacynode_peerroster_reachable_peers_per_asn gauge
yacynode_peerroster_reachable_peers_per_asn{asn="64500"} 2
yacynode_peerroster_reachable_peers_per_asn{asn="64501"} 1
yacynode_peerroster_reachable_peers_per_asn{asn="64502"} 1
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_peerroster_reachable_peers_per_asn",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
