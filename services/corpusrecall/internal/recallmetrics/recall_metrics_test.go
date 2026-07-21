package recallmetrics

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
)

func TestRecallMetricsRecordsAndExposesCounters(t *testing.T) {
	metrics := New()

	metrics.RequestAccepted()
	metrics.RequestRejected()
	metrics.RepresentationRecalled(pagerecall.Kind("markdown"))
	metrics.RepresentationUnavailable(pagerecall.Kind("text"))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"corpusrecall_requests_accepted_total 1",
		"corpusrecall_requests_rejected_total 1",
		`corpusrecall_representations_recalled_total{kind="markdown"} 1`,
		`corpusrecall_representations_unavailable_total{kind="text"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q, got:\n%s", want, body)
		}
	}
}
