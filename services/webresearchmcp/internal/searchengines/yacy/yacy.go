// Package yacy reads YaCy's public /yacysearch.json search interface, which the operator's
// yacydhtsearch service serves for the YaCy network.
package yacy

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
	searchPath              = "/yacysearch.json"
	queryField              = "query"
	searchResponseByteLimit = 8 << 20
)

type YaCy struct {
	yacyURL string
	client  *http.Client
}

func NewYaCy(yacyURL string, searchDeadline time.Duration) *YaCy {
	return &YaCy{
		yacyURL: yacyURL,
		client:  &http.Client{Timeout: searchDeadline},
	}
}

func (e *YaCy) SearchResultsFor(
	ctx context.Context,
	query string,
) ([]websearch.SearchResult, error) {
	request, err := e.searchRequestFor(ctx, query)
	if err != nil {
		return nil, err
	}
	response, err := e.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ask yacy for %q: %w", query, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yacy answered %s for %q", response.Status, query)
	}
	return searchResultsIn(response)
}

func (e *YaCy) searchRequestFor(
	ctx context.Context,
	query string,
) (*http.Request, error) {
	searchURL := e.yacyURL + searchPath + "?" + url.Values{queryField: {query}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build the yacy request for %q: %w", query, err)
	}
	return request, nil
}

type searchPage struct {
	Channels []struct {
		Items []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Description string `json:"description"`
		} `json:"items"`
	} `json:"channels"`
}

func searchResultsIn(response *http.Response) ([]websearch.SearchResult, error) {
	var page searchPage
	body := io.LimitReader(response.Body, searchResponseByteLimit)
	if err := json.NewDecoder(body).Decode(&page); err != nil {
		return nil, fmt.Errorf("read the yacy answer: %w", err)
	}
	var results []websearch.SearchResult
	for _, channel := range page.Channels {
		for _, item := range channel.Items {
			results = append(results, websearch.SearchResult{
				URL:     item.Link,
				Title:   item.Title,
				Snippet: item.Description,
			})
		}
	}
	return results, nil
}
