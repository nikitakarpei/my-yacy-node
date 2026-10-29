package metrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerport"
)

func TestPeerPortCountsConnectionsPerScheme(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := metrics.NewPeerPortMetrics(registry)

	observer.ObserveConnection(peerport.HTTPSScheme)
	observer.ObserveConnection(peerport.HTTPSScheme)

	expected := `
# HELP yacynode_peerport_connections_total Connections to the peer port, by their scheme.
# TYPE yacynode_peerport_connections_total counter
yacynode_peerport_connections_total{scheme="http"} 0
yacynode_peerport_connections_total{scheme="https"} 2
`
	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"yacynode_peerport_connections_total",
	); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
