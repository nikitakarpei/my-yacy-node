package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	peerjudgementsobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementsobservers/prometheus"
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

func peerAtVersion() peerjudgements.PeerAtVersion {
	return peerjudgements.PeerAtVersion{
		Peer:    yacymodel.WordHash("peer"),
		Version: "yacy_v1.925",
	}
}

func judgedPeer(judgement peerjudgements.Judgement) peerjudgements.JudgedPeer {
	return peerjudgements.JudgedPeer{PeerAtVersion: peerAtVersion(), Judgement: judgement}
}

func peerStanding(standing peerjudgements.Standing) peerjudgements.PeerStanding {
	return peerjudgements.PeerStanding{PeerAtVersion: peerAtVersion(), Standing: standing}
}

func carriedBy(t *testing.T, body string, published []string) {
	t.Helper()

	for _, series := range published {
		if !strings.Contains(body, series) {
			t.Fatalf("metrics do not carry %q:\n%s", series, body)
		}
	}
}

func TestTheStandingOfAPeerIsCountedUnderTheFormItStoodOn(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerjudgementsobserversprometheus.New(registry)

	metrics.PeerStood(
		t.Context(), peerjudgements.NamedDocuments, peerStanding(peerjudgements.Ignoring),
	)

	carriedBy(t, publishedBy(t, registry), []string{
		`yacydhtsearch_peer_standings_total{form="named documents",standing="ignoring"} 1`,
		`yacydhtsearch_peer_standings_total{form="named documents",standing="honoring"} 0`,
	})
}

func TestTheJudgementOfAPeerIsCountedUnderTheFormItWasJudgedOn(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := peerjudgementsobserversprometheus.New(registry)

	metrics.PeerJudged(
		t.Context(), peerjudgements.NamedDocuments, judgedPeer(peerjudgements.Honored),
	)

	carriedBy(t, publishedBy(t, registry), []string{
		`yacydhtsearch_peer_judgements_total{form="named documents",judged="honored"} 1`,
		`yacydhtsearch_peer_judgements_total{form="named documents",judged="ignored"} 0`,
	})
}

func TestEveryStandingAndEveryJudgementIsPublishedFromTheStart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	peerjudgementsobserversprometheus.New(registry)

	carriedBy(t, publishedBy(t, registry), []string{
		`yacydhtsearch_peer_standings_total{form="named documents",standing="honoring"} 0`,
		`yacydhtsearch_peer_standings_total{form="named documents",standing="ignoring"} 0`,
		`yacydhtsearch_peer_standings_total{form="named documents",standing="never judged"} 0`,
		`yacydhtsearch_peer_standings_total{form="named documents",standing="version changed"} 0`,
		`yacydhtsearch_peer_standings_total{form="named documents",standing="interval passed"} 0`,
		`yacydhtsearch_peer_judgements_total{form="named documents",judged="honored"} 0`,
		`yacydhtsearch_peer_judgements_total{form="named documents",judged="ignored"} 0`,
		`yacydhtsearch_peer_judgements_total{form="named documents",judged="no evidence"} 0`,
	})
}
