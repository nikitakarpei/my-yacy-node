package prometheus_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	peerjudgementledgersobserversjetstreamprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgersobservers/jetstream/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const question peerjudgements.Question = "a question about the peer"

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestEachFailureIsPublishedUnderItsAction(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerjudgementledgersobserversjetstreamprometheus.New(registry)
	peer := yacymodel.WordHash("one")
	refused := errors.New("bucket refused")

	metrics.JudgementDropped(t.Context(), peer, question)
	metrics.JudgementHoldFailed(t.Context(), peer, question, refused)
	metrics.WatchFailed(t.Context(), refused)
	metrics.WatchEnded(t.Context())
	metrics.JudgementUndecodable(t.Context(), "key", refused)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="queue"} 1`,
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="hold"} 1`,
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="watch"} 2`,
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="decode"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryJudgementLedgerFailureIsPublishedBeforeTheFirstFailure(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	peerjudgementledgersobserversjetstreamprometheus.New(registry)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="queue"} 0`,
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="hold"} 0`,
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="watch"} 0`,
		`yacydhtsearch_peer_judgement_ledger_failures_total{action="decode"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}
