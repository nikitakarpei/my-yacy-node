package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	peerdirectoryobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestEveryDirectoryChangeIsCountedUnderItsName(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerdirectoryobserversprometheus.New(registry)
	peer := yacymodel.WordHash("peer")

	metrics.PeerAdmitted(t.Context(), peer, 2)
	metrics.PeerAnswered(t.Context(), peer, "http://peer.example:8090", time.Now())
	metrics.PeerWentSilent(t.Context(), peer)
	metrics.PeerDropped(t.Context(), peer)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_directory_peer_changes_total{change="admitted"} 1`,
		`yacydhtsearch_directory_peer_changes_total{change="answered"} 1`,
		`yacydhtsearch_directory_peer_changes_total{change="wentSilent"} 1`,
		`yacydhtsearch_directory_peer_changes_total{change="dropped"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestHowFullTheDirectoryIsIsPublishedAgainstItsCapacity(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerdirectoryobserversprometheus.New(registry)

	metrics.PeersKnown(t.Context(), 12, 5, 4096)

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_directory_peers 12") ||
		!strings.Contains(body, "yacydhtsearch_directory_answering_peers 5") ||
		!strings.Contains(body, "yacydhtsearch_directory_capacity 4096") {
		t.Fatalf("metrics do not carry the directory fill:\n%s", body)
	}
}

func TestEveryDirectoryChangeIsPublishedBeforeTheFirstChange(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	peerdirectoryobserversprometheus.New(registry)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_directory_peer_changes_total{change="admitted"} 0`,
		`yacydhtsearch_directory_peer_changes_total{change="answered"} 0`,
		`yacydhtsearch_directory_peer_changes_total{change="wentSilent"} 0`,
		`yacydhtsearch_directory_peer_changes_total{change="dropped"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
