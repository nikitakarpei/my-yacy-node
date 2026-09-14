package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	peermatchedobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peermatchedobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
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

func TestOnePeerMatchedSpreadPublishesHowManyPeersAnsweredAndHowLongItTook(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peermatchedobserversprometheus.New(registry, queryBudget)

	metrics.PeerMatchedSpreadPerformed(t.Context(), peermatched.PerformedPeerMatchedSpread{
		AmountOfQueryWords:              3,
		AmountOfPeersAsked:              8,
		AmountOfPeersThatAnswered:       4,
		AmountOfPeersThatMatchedNothing: 3,
		TimeSpent:                       250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_peer_matched_spread_answering_peers_ratio_sum 0.5",
		"yacydhtsearch_peer_matched_spread_duration_seconds_sum 0.25",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASpreadNoPeerAnsweredPublishesAnAnsweringRatioOfNone(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peermatchedobserversprometheus.New(registry, queryBudget)

	metrics.PeerMatchedSpreadPerformed(t.Context(), peermatched.PerformedPeerMatchedSpread{
		AmountOfQueryWords: 1,
		AmountOfPeersAsked: 4,
		TimeSpent:          time.Second,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_peer_matched_spread_answering_peers_ratio_sum 0") {
		t.Fatalf("metrics do not carry an answering peers ratio of none:\n%s", body)
	}
}

func TestASpreadThatAskedNoPeerPublishesNoAnsweringRatio(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peermatchedobserversprometheus.New(registry, queryBudget)

	metrics.PeerMatchedSpreadPerformed(t.Context(), peermatched.PerformedPeerMatchedSpread{
		AmountOfQueryWords: 1,
		TimeSpent:          time.Second,
	})

	body := publishedBy(t, registry)
	if !strings.Contains(body, "yacydhtsearch_peer_matched_spread_answering_peers_ratio_count 0") {
		t.Fatalf("metrics carry an answering peers ratio for a spread with no peer:\n%s", body)
	}
}
