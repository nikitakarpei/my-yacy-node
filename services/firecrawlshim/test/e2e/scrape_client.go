//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type scrapeResult struct {
	Success bool `json:"success"`
	Data    struct {
		Markdown string `json:"markdown"`
		Metadata struct {
			SourceURL string `json:"sourceURL"`
		} `json:"metadata"`
	} `json:"data"`
	Error string `json:"error"`
}

func scrape(
	t *testing.T, ctx context.Context, baseURL, url string,
) (scrapeResult, int) {
	t.Helper()
	body := `{"url":"` + url + `","formats":["markdown"]}`
	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, baseURL+"/v1/scrape", strings.NewReader(body),
	)
	if err != nil {
		t.Fatalf("build scrape request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("scrape %q: %v", url, err)
	}
	defer func() { _ = response.Body.Close() }()

	var result scrapeResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode scrape response: %v", err)
	}
	return result, response.StatusCode
}
