package webuistyle_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

func TestStylesheetServesTheLookOfThePages(t *testing.T) {
	answer := httptest.NewRecorder()

	webuistyle.Stylesheet().ServeHTTP(
		answer,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, webuistyle.StylesheetPath, nil),
	)

	if answer.Code != http.StatusOK ||
		!strings.HasPrefix(answer.Header().Get("Content-Type"), "text/css") ||
		!strings.Contains(answer.Body.String(), "--webui-accent") {
		t.Errorf(
			"answer = %d %q, want the stylesheet",
			answer.Code,
			answer.Header().Get("Content-Type"),
		)
	}
}
