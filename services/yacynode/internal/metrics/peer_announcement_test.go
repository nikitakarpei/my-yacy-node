package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerannouncement"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

func TestPeerAnnouncementShowsReachablePeersPerSchemeOfTheLastContacts(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewPeerAnnouncementMetrics(registry)

	observer.ObservePeerContacts(
		map[peerannouncement.ContactOutcome]int{},
		map[peerwire.Scheme]int{peerwire.HTTP: 4, peerwire.HTTPS: 1},
	)
	observer.ObservePeerContacts(
		map[peerannouncement.ContactOutcome]int{},
		map[peerwire.Scheme]int{peerwire.HTTP: 3, peerwire.HTTPS: 2},
	)

	expected := `
# HELP yacynode_peerannouncement_reachable_peers_per_scheme Peers confirmed reachable when this node last contacted its peers, by the scheme of the origin that answered the greet.
# TYPE yacynode_peerannouncement_reachable_peers_per_scheme gauge
yacynode_peerannouncement_reachable_peers_per_scheme{scheme="http"} 3
yacynode_peerannouncement_reachable_peers_per_scheme{scheme="https"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_peerannouncement_reachable_peers_per_scheme",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
