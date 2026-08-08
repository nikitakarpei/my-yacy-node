//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

type indexedPage struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

func assertIndexedPage(t *testing.T, page indexedPage, originURL string) {
	t.Helper()
	if page.Title != originTitle {
		t.Errorf("indexed title = %q, want %q", page.Title, originTitle)
	}
	if !strings.Contains(page.Content, originBody) {
		t.Errorf("indexed content = %q, want it to contain %q", page.Content, originBody)
	}
	if page.Language != indexedLanguage {
		t.Errorf("indexed language = %q, want %q", page.Language, indexedLanguage)
	}
	if !strings.Contains(page.URL, strings.TrimSuffix(originURL, "/")) {
		t.Errorf("indexed url = %q, want it to carry %q", page.URL, originURL)
	}
}
