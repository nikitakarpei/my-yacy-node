package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	peerpresenceobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresenceobservers/jetstream/prometheus"
)

var errSnapshot = errors.New("the bucket did not answer")

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestEverySnapshotFailureIsPublishedApartFromTheSnapshotsWritten(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerpresenceobserversjetstreamprometheus.New(registry)

	metrics.SnapshotsReadFailed(t.Context(), errSnapshot)
	metrics.SnapshotUndecodable(t.Context(), "aaaaaaaaaaaa.bbb", errSnapshot)
	metrics.SnapshotWriteFailed(t.Context(), "aaaaaaaaaaaa.bbb", errSnapshot)
	metrics.PeersSnapshotted(t.Context(), 12)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_presence_snapshot_failures_total{failure="snapshotsUnread"} 1`,
		`yacydhtsearch_peer_presence_snapshot_failures_total{failure="snapshotUndecodable"} 1`,
		`yacydhtsearch_peer_presence_snapshot_failures_total{failure="snapshotUnwritten"} 1`,
		"yacydhtsearch_peer_presence_snapshots_written_total 12",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
