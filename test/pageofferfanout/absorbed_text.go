//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

type indexedPage struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

type manticoreHits struct {
	Hits struct {
		Hits []struct {
			Source indexedPage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func absorbedTextOf(t *testing.T, ctx context.Context, manticoreURL, term string) indexedPage {
	t.Helper()
	var found indexedPage
	indexed := pollwait.For(absorptionDeadline, func() bool {
		page, matched := manticoreMatchOnce(t, ctx, manticoreURL, term)
		if !matched {
			return false
		}
		found = page
		return true
	})
	if !indexed {
		t.Fatalf(
			"corpustext never indexed %q into %s within %s",
			term, manticoreTable, absorptionDeadline,
		)
	}
	return found
}

func manticoreMatchOnce(
	t *testing.T,
	ctx context.Context,
	manticoreURL, term string,
) (indexedPage, bool) {
	t.Helper()
	query, err := json.Marshal(map[string]any{
		"table": manticoreTable,
		"query": map[string]any{"match": map[string]any{"content": term}},
	})
	if err != nil {
		t.Fatalf("marshal manticore query: %v", err)
	}
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, manticoreURL+"/search", bytes.NewReader(query),
	)
	if err != nil {
		t.Fatalf("build manticore search request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return indexedPage{}, false
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return indexedPage{}, false
	}
	var hits manticoreHits
	if err := json.NewDecoder(response.Body).Decode(&hits); err != nil {
		return indexedPage{}, false
	}
	if len(hits.Hits.Hits) == 0 {
		return indexedPage{}, false
	}
	return hits.Hits.Hits[0].Source, true
}
