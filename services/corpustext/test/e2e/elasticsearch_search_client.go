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

type elasticsearchHits struct {
	Hits struct {
		Hits []struct {
			Source indexedPage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func waitForElasticsearchContentHit(
	t *testing.T,
	ctx context.Context,
	elasticsearchURL, target, term string,
) indexedPage {
	t.Helper()
	var found indexedPage
	ok := pollwait.For(30*time.Second, func() bool {
		page, ok := elasticsearchSearchOnce(t, ctx, elasticsearchURL, target, term)
		if !ok {
			return false
		}
		found = page
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
) (indexedPage, bool) {
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
		return indexedPage{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return indexedPage{}, false
	}
	var body elasticsearchHits
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return indexedPage{}, false
	}
	if len(body.Hits.Hits) == 0 {
		return indexedPage{}, false
	}
	return body.Hits.Hits[0].Source, true
}
