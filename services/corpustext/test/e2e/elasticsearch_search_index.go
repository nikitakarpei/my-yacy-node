//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/elasticsearch"
)

const (
	elasticsearchAlias         = "elasticsearch"
	elasticsearchIndexBase     = "yacy_text"
	elasticsearchIndexPrefix   = elasticsearchIndexBase + "_v1"
	elasticsearchLanguageIndex = elasticsearchIndexPrefix + "_" + indexedLanguage
	elasticsearchIndexPattern  = elasticsearchIndexPrefix + "_*"
)

func startElasticsearch(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return elasticsearch.Start(t, ctx, networkName, elasticsearchAlias)
}

func elasticsearchNetworkURL() string {
	return elasticsearch.NetworkURL(elasticsearchAlias)
}

func elasticsearchCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE":  "elasticsearch",
		"ELASTICSEARCH_URL":    elasticsearchNetworkURL(),
		"ELASTICSEARCH_INDEX":  elasticsearchIndexBase,
		"CORPUSTEXT_LANGUAGES": indexedLanguage,
	}
}
