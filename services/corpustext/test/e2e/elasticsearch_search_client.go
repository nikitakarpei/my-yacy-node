//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

type searchHit struct {
	Source struct {
		Title    string `json:"title"`
		URL      string `json:"url"`
		Content  string `json:"content"`
		Language string `json:"language"`
	} `json:"_source"`
}

type searchResponse struct {
	Hits struct {
		Hits []searchHit `json:"hits"`
	} `json:"hits"`
}

func waitForElasticsearchContentHit(
	t *testing.T,
	ctx context.Context,
	elasticsearchURL, target, term string,
) searchHit {
	t.Helper()
	var found searchHit
	ok := pollwait.For(30*time.Second, func() bool {
		hit, ok := elasticsearchSearchOnce(t, ctx, elasticsearchURL, target, term)
		if !ok {
			return false
		}
		found = hit
		return true
	})
	if !ok {
		t.Fatalf("elasticsearch never matched %q in %s", term, target)
	}
	return found
}

func elasticsearchSearchOnce(
	t *testing.T,
	ctx context.Context,
	elasticsearchURL, target, term string,
) (searchHit, bool) {
	t.Helper()
	query, err := json.Marshal(map[string]any{
		"query": map[string]any{"match": map[string]any{"content": term}},
	})
	if err != nil {
		t.Fatalf("marshal elasticsearch query: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		elasticsearchURL+"/"+target+"/_search",
		bytes.NewReader(query),
	)
	if err != nil {
		t.Fatalf("build search request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return searchHit{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return searchHit{}, false
	}
	var body searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return searchHit{}, false
	}
	if len(body.Hits.Hits) == 0 {
		return searchHit{}, false
	}
	return body.Hits.Hits[0], true
}
