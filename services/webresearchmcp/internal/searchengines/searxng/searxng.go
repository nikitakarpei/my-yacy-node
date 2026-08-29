// Package searxng reads the JSON search interface of one SearXNG instance. It asks SearXNG
// to leave every result link as the result carries it, so an answer points a caller at the
// page itself and not at a link that routes the caller through the operator's stack.
package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/websearch"
)

const (
	searchPath              = "/search"
	jsonSearchFormat        = "json"
	resultLinkRouterPlugin  = "result_link_router"
	searchResponseByteLimit = 8 << 20
)

type SearXNG struct {
	searxngURL string
	client     *http.Client
}

func NewSearXNG(searxngURL string, searchDeadline time.Duration) *SearXNG {
	return &SearXNG{
		searxngURL: searxngURL,
		client:     &http.Client{Timeout: searchDeadline},
	}
}

func (e *SearXNG) SearchResultsFor(
	ctx context.Context,
	query string,
) ([]websearch.SearchResult, error) {
	request, err := e.searchRequestFor(ctx, query)
	if err != nil {
		return nil, err
	}
	response, err := e.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ask searxng for %q: %w", query, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searxng answered %s for %q", response.Status, query)
	}
	return searchResultsIn(response)
}

func (e *SearXNG) searchRequestFor(
	ctx context.Context,
	query string,
) (*http.Request, error) {
	searchURL := e.searxngURL + searchPath + "?" + url.Values{
		"q":                {query},
		"format":           {jsonSearchFormat},
		"disabled_plugins": {resultLinkRouterPlugin},
	}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build the searxng request for %q: %w", query, err)
	}
	return request, nil
}

type searchAnswer struct {
	Results []struct {
		URL     string `json:"url"`
		Title   string `json:"title"`
		Content string `json:"content"`
	} `json:"results"`
}

func searchResultsIn(response *http.Response) ([]websearch.SearchResult, error) {
	var answer searchAnswer
	body := io.LimitReader(response.Body, searchResponseByteLimit)
	if err := json.NewDecoder(body).Decode(&answer); err != nil {
		return nil, fmt.Errorf("read the searxng answer: %w", err)
	}
	results := make([]websearch.SearchResult, 0, len(answer.Results))
	for _, found := range answer.Results {
		results = append(results, websearch.SearchResult{
			URL:     found.URL,
			Title:   found.Title,
			Snippet: found.Content,
		})
	}
	return results, nil
}
