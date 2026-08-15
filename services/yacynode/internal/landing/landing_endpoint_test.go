package landing_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/landing"
)

func TestLandingEndpointServesHTML(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	landing.NewEndpoint().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("content type = %q, want %q", got, "text/html; charset=utf-8")
	}
	body := rec.Body.String()
	for _, want := range []string{"alpha", "RWI", "github.com/nikitakarpei/yacy-rwi-node/issues"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestLandingEndpointRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil)

	landing.NewEndpoint().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("allow = %q, want %q", got, http.MethodGet)
	}
}
