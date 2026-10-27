package pageshell_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageshell"
)

func TestStylesheetServesTheLookOfThePages(t *testing.T) {
	answer := httptest.NewRecorder()

	pageshell.Stylesheet().ServeHTTP(
		answer,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, pageshell.StylesheetPath, nil),
	)

	if answer.Code != http.StatusOK ||
		!strings.HasPrefix(answer.Header().Get("Content-Type"), "text/css") ||
		!strings.Contains(answer.Body.String(), "--shell-accent") {
		t.Errorf(
			"answer = %d %q, want the stylesheet",
			answer.Code,
			answer.Header().Get("Content-Type"),
		)
	}
}
