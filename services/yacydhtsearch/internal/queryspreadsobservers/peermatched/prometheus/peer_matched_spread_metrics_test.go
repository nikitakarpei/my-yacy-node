package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	queryspreadsobserverspeermatchedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/peermatched/prometheus"
)

const queryBudget = 5 * time.Second

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestOnePeerMatchedSpreadPublishesHowLongItTook(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverspeermatchedprometheus.New(registry, queryBudget)

	metrics.PeerMatchedSpreadPerformed(t.Context(), peermatched.PerformedPeerMatchedSpread{
		AmountOfQueryWords:              3,
		AmountOfPeersThatMatchedNothing: 3,
		TimeSpent:                       250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_peer_matched_spread_duration_seconds_sum 0.25") {
		t.Fatalf("metrics do not carry the duration of the spread:\n%s", body)
	}
}
