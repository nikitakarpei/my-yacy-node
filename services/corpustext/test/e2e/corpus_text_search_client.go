//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

type corpusTextMatches struct {
	Matches []corpusTextMatch `json:"matches"`
}

type corpusTextMatch struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

func waitForCorpusTextMatch(
	t *testing.T,
	ctx context.Context,
	searchURL string,
) corpusTextMatch {
	t.Helper()
	var found corpusTextMatch
	ok := pollwait.For(30*time.Second, func() bool {
		match, ok := corpusTextSearchOnce(ctx, searchURL, stemmedTerm)
		if !ok {
			return false
		}
		found = match
		return true
	})
	if !ok {
		t.Fatalf("corpustext never matched %q", stemmedTerm)
	}
	return found
}

func corpusTextSearchOnce(
	ctx context.Context,
	searchURL, term string,
) (corpusTextMatch, bool) {
	target := searchURL + "/search?" + url.Values{
		"query":    {term},
		"language": {indexedLanguage},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return corpusTextMatch{}, false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return corpusTextMatch{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return corpusTextMatch{}, false
	}
	var body corpusTextMatches
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return corpusTextMatch{}, false
	}
	if len(body.Matches) == 0 {
		return corpusTextMatch{}, false
	}
	return body.Matches[0], true
}

func assertCorpusTextMatch(t *testing.T, match corpusTextMatch, originURL string) {
	t.Helper()
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
