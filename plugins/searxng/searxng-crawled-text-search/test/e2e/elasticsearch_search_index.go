//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/elasticsearch"
)

const (
	elasticsearchAlias         = "elasticsearch"
	elasticsearchIndexBase     = "yacy_text"
	elasticsearchIndexPrefix   = elasticsearchIndexBase + "_v1"
	elasticsearchIndexPattern  = elasticsearchIndexPrefix + "_*"
	elasticsearchCatchAllIndex = elasticsearchIndexPrefix + "_und"
)

func startElasticsearch(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return elasticsearch.Start(t, ctx, networkName, elasticsearchAlias)
}

func awaitElasticsearchCorpus(t *testing.T, ctx context.Context, elasticsearchURL string) {
	t.Helper()
	awaitIndexedCorpus(t, func() int {
		return documentsInElasticsearchIndex(t, ctx, elasticsearchURL, elasticsearchIndexPattern)
	})
}

type elasticsearchHits struct {
	Hits struct {
		Hits []struct{} `json:"hits"`
	} `json:"hits"`
}

func documentsInElasticsearchIndex(
	t *testing.T,
	ctx context.Context,
	elasticsearchURL, index string,
) int {
	t.Helper()
	query, err := json.Marshal(
		map[string]any{"query": map[string]any{"match_all": map[string]any{}}},
	)
	if err != nil {
		t.Fatalf("marshal elasticsearch query: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		elasticsearchURL+"/"+index+"/_search",
		bytes.NewReader(query),
	)
	if err != nil {
		t.Fatalf("build elasticsearch search request: %v", err)
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
	var hits elasticsearchHits
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return 0
	}
	return len(hits.Hits.Hits)
}

func elasticsearchCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE": "elasticsearch",
		"ELASTICSEARCH_URL":   elasticsearch.NetworkURL(elasticsearchAlias),
		"ELASTICSEARCH_INDEX": elasticsearchIndexBase,
	}
}

func elasticsearchEngineSettings() string {
	return "    search_index_engine: elasticsearch\n" +
		"    elasticsearch_url: " + elasticsearch.NetworkURL(elasticsearchAlias) + "\n" +
		"    elasticsearch_index: " + elasticsearchIndexPrefix + "\n"
}
