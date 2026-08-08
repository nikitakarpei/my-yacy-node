package prometheus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	progressobserversprometheus "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/progressobservers/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall/pagerecall"
)

func TestExpositionReportsEveryRecallOutcomeObserved(t *testing.T) {
	metrics := progressobserversprometheus.NewRecallMetrics()

	metrics.RequestAccepted()
	metrics.RequestRejected()
	metrics.RepresentationRecalled(pagerecall.RepresentationKind("markdown"))
	metrics.RepresentationUnavailable(pagerecall.RepresentationKind("text"))

	recorder := httptest.NewRecorder()
	metrics.Exposition().ServeHTTP(recorder, httptest.NewRequestWithContext(
		context.Background(), http.MethodGet, "/metrics", nil,
	))

	body := recorder.Body.String()
	for _, want := range []string{
		"corpusrecall_requests_accepted_total 1",
		"corpusrecall_requests_rejected_total 1",
		`corpusrecall_representations_recalled_total{kind="markdown"} 1`,
		`corpusrecall_representations_unavailable_total{kind="text"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("exposition missing %q, got:\n%s", want, body)
		}
	}
}
