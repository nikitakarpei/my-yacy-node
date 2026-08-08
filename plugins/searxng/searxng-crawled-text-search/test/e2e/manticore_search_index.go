//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/manticore"
)

const (
	manticoreAlias         = "manticore"
	manticoreTableBase     = "yacy_text"
	manticoreFanOutTable   = manticoreTableBase + "_v1"
	manticoreCatchAllTable = manticoreFanOutTable + "_und"
)

func startManticore(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	return manticore.Start(t, ctx, networkName, manticoreAlias)
}

func awaitManticoreCorpus(t *testing.T, ctx context.Context, manticoreURL string) {
	t.Helper()
	awaitIndexedCorpus(t, func() int {
		return documentsInManticoreTable(t, ctx, manticoreURL, manticoreFanOutTable)
	})
}

type manticoreHits struct {
	Hits struct {
		Hits []struct{} `json:"hits"`
	} `json:"hits"`
}

func documentsInManticoreTable(
	t *testing.T,
	ctx context.Context,
	manticoreURL, table string,
) int {
	t.Helper()
	query, err := json.Marshal(map[string]any{
		"table": table,
		"query": map[string]any{"match_all": map[string]any{}},
	})
	if err != nil {
		t.Fatalf("marshal manticore query: %v", err)
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, manticoreURL+"/search", bytes.NewReader(query),
	)
	if err != nil {
		t.Fatalf("build manticore search request: %v", err)
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
	var hits manticoreHits
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return 0
	}
	return len(hits.Hits.Hits)
}

func manticoreCorpusTextEnv() map[string]string {
	return map[string]string{
		"SEARCH_INDEX_ENGINE": "manticore",
		"MANTICORE_URL":       manticore.NetworkURL(manticoreAlias),
		"MANTICORE_TABLE":     manticoreTableBase,
	}
}

func manticoreEngineSettings() string {
	return "    search_index_engine: manticore\n" +
		"    manticore_url: " + manticore.NetworkURL(manticoreAlias) + "\n" +
		"    manticore_table: " + manticoreFanOutTable + "\n"
}
