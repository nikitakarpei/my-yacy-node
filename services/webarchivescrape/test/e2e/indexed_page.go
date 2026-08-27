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

type (
	indexedPage struct {
		URL     string `json:"url"`
		Content string `json:"content"`
	}
	manticoreHits struct {
		Hits struct {
			Hits []struct {
				Source indexedPage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
)

func indexedPageContaining(
	t *testing.T,
	ctx context.Context,
	manticoreURL, content string,
) indexedPage {
	t.Helper()
	var page indexedPage
	if !pollwait.For(
		30*time.Second,
		func() bool { page = manticoreMatch(t, ctx, manticoreURL, content); return page.URL != "" },
	) {
		t.Fatalf("corpustext did not index content %q", content)
	}
	return page
}

func manticoreMatch(t *testing.T, ctx context.Context, manticoreURL, content string) indexedPage {
	t.Helper()
	query, _ := json.Marshal(
		map[string]any{
			"table": manticoreTable,
			"query": map[string]any{"match": map[string]string{"content": content}},
		},
	)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		manticoreURL+"/search",
		bytes.NewReader(query),
	)
	if err != nil {
		t.Fatalf("build manticore query: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return indexedPage{}
	}
	defer func() { _ = response.Body.Close() }()
	var hits manticoreHits
	if response.StatusCode != http.StatusOK ||
		json.NewDecoder(response.Body).Decode(&hits) != nil ||
		len(hits.Hits.Hits) == 0 {
		return indexedPage{}
	}
	return hits.Hits.Hits[0].Source
}
