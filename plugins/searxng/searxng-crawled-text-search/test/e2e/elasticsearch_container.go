//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/elasticsearch"
)

const elasticsearchAlias = "elasticsearch"

func startElasticsearch(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return elasticsearch.Start(t, ctx, networkName, elasticsearchAlias)
}

func awaitElasticsearchCorpus(t *testing.T, ctx context.Context, elasticsearchURL string) {
	t.Helper()
	awaitIndexedCorpus(
		t,
		ctx,
		elasticsearchURL+"/"+fanOutPrefix+"_*/_search",
		map[string]any{"query": map[string]any{"match_all": map[string]any{}}},
	)
}

func elasticsearchCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE": "elasticsearch",
		"ELASTICSEARCH_URL":   elasticsearch.NetworkURL(elasticsearchAlias),
		"ELASTICSEARCH_INDEX": indexBaseName,
	}
}

func elasticsearchEngineSettings() string {
	return "    search_index_engine: elasticsearch\n" +
		"    elasticsearch_url: " + elasticsearch.NetworkURL(elasticsearchAlias) + "\n" +
		"    elasticsearch_index: " + fanOutPrefix + "\n"
}
