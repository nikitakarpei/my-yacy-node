// Package yacysearch reads YaCy's public /yacysearch.json search interface of one search
// service, such as yacydhtsearch. A result whose publication date does not parse comes back
// without one.
package yacysearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/searchpages"
)

const (
	searchPath            = "/yacysearch.json"
	searchAnswerByteLimit = 8 << 20
	publicationLayout     = time.RFC1123Z
)

type YaCySearch struct {
	searchURL string
	client    *http.Client
}

func New(searchURL *url.URL, callDeadline time.Duration) *YaCySearch {
	return &YaCySearch{
		searchURL: strings.TrimSuffix(searchURL.String(), "/"),
		client:    &http.Client{Timeout: callDeadline},
	}
}

func (s *YaCySearch) SearchResultsFor(
	ctx context.Context,
	query string,
	offset int,
	limit int,
) ([]searchpages.SearchResult, error) {
	request, err := s.searchRequestFor(ctx, query, offset, limit)
	if err != nil {
		return nil, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ask %s for %q: %w", s.searchURL, query, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %s for %q", s.searchURL, response.Status, query)
	}
	return searchResultsIn(response)
}

func (s *YaCySearch) searchRequestFor(
	ctx context.Context,
	query string,
	offset int,
	limit int,
) (*http.Request, error) {
	searchURL := s.searchURL + searchPath + "?" + url.Values{
		"query":          {query},
		"startRecord":    {strconv.Itoa(offset)},
		"maximumRecords": {strconv.Itoa(limit)},
	}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build the search request for %q: %w", query, err)
	}
	return request, nil
}

type searchAnswer struct {
	Channels []struct {
		Items []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Description string `json:"description"`
			PubDate     string `json:"pubDate"`
		} `json:"items"`
	} `json:"channels"`
}

func searchResultsIn(response *http.Response) ([]searchpages.SearchResult, error) {
	var answer searchAnswer
	body := io.LimitReader(response.Body, searchAnswerByteLimit)
	if err := json.NewDecoder(body).Decode(&answer); err != nil {
		return nil, fmt.Errorf("read the search answer: %w", err)
	}
	var results []searchpages.SearchResult
	for _, channel := range answer.Channels {
		for _, item := range channel.Items {
			results = append(results, searchpages.SearchResult{
				Title:       item.Title,
				Address:     item.Link,
				Description: item.Description,
				PublishedAt: publicationOf(item.PubDate),
			})
		}
	}
	return results, nil
}

func publicationOf(pubDate string) time.Time {
	published, err := time.Parse(publicationLayout, pubDate)
	if err != nil {
		return time.Time{}
	}
	return published
}
