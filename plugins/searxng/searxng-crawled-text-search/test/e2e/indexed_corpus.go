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

const indexingLimit = 60 * time.Second

type indexedHits struct {
	Hits struct {
		Hits []struct{} `json:"hits"`
	} `json:"hits"`
}

func awaitIndexedCorpus(t *testing.T, ctx context.Context, searchURL string, query any) {
	t.Helper()
	body, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("marshal readiness query: %v", err)
	}
	ok := pollwait.For(indexingLimit, func() bool {
		return indexedDocumentsIn(t, ctx, searchURL, body) >= len(crawledPages())
	})
	if !ok {
		t.Fatal("the crawled corpus was never indexed")
	}
}

func indexedDocumentsIn(t *testing.T, ctx context.Context, searchURL string, body []byte) int {
	t.Helper()
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, searchURL, bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("build readiness request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var hits indexedHits
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return 0
	}
	return len(hits.Hits.Hits)
}
