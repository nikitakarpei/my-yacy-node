//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/corpustextservice"
)

const indexedLanguage = "en"

func assertCorpusTextFinds(
	t *testing.T,
	ctx context.Context,
	corpusText corpustextservice.CorpusText,
	originURL string,
) {
	t.Helper()
	match := corpusText.FirstMatchFor(t, ctx, stemmedTerm, indexedLanguage)
	if match.Title != originTitle {
		t.Errorf("matched title = %q, want %q", match.Title, originTitle)
	}
	if !strings.Contains(match.Snippet, stemmedTerm) {
		t.Errorf("matched snippet = %q, want it to carry %q", match.Snippet, stemmedTerm)
	}
	if !strings.Contains(match.URL, strings.TrimSuffix(originURL, "/")) {
		t.Errorf("matched url = %q, want it to carry %q", match.URL, originURL)
	}
}
