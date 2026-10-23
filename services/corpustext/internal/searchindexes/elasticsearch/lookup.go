package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

type Lookup struct {
	endpoint string
	indexes  languageindex.LanguageIndexes
	client   *http.Client
}

func NewLookup(
	endpoint string,
	indexes languageindex.LanguageIndexes,
	client *http.Client,
) *Lookup {
	return &Lookup{
		endpoint: strings.TrimRight(endpoint, "/"),
		indexes:  indexes,
		client:   client,
	}
}

func (lookup *Lookup) MatchesFor(
	ctx context.Context,
	search textsearch.Search,
) ([]searchdocument.PageMatch, error) {
	body, err := json.Marshal(searchRequestFor(search))
	if err != nil {
		return nil, fmt.Errorf("marshal search request: %w", err)
	}
	response, err := lookup.searchResponseTo(ctx, body)
	if err != nil {
		return nil, err
	}
	return response.pageMatches(), nil
}

type searchRequest struct {
	From      int            `json:"from"`
	Size      int            `json:"size"`
	Query     map[string]any `json:"query"`
	Highlight map[string]any `json:"highlight"`
	Source    []string       `json:"_source"`
}

func searchRequestFor(search textsearch.Search) searchRequest {
	return searchRequest{
		From:  search.Offset,
		Size:  search.Limit,
		Query: queryFor(search),
		Highlight: map[string]any{
			"fields": map[string]any{
				"content": map[string]any{
					"fragment_size":       textsearch.SnippetLength,
					"number_of_fragments": 1,
					"no_match_size":       textsearch.SnippetLength,
					"order":               "score",
				},
			},
			"pre_tags":  []string{""},
			"post_tags": []string{""},
		},
		Source: []string{"title", "url"},
	}
}

func queryFor(search textsearch.Search) map[string]any {
	everyWord := map[string]any{
		"combined_fields": map[string]any{
			"query": search.Query,
			"fields": []string{
				fmt.Sprintf("title^%d", textsearch.TitleWeight),
				fmt.Sprintf("content^%d", textsearch.ContentWeight),
				fmt.Sprintf("auxiliary_text^%d", textsearch.AuxiliaryTextWeight),
			},
			"operator": "and",
		},
	}
	if search.Language == "" {
		return everyWord
	}
	return map[string]any{
		"bool": map[string]any{
			"must":   []any{everyWord},
			"filter": []any{map[string]any{"term": map[string]any{"language": search.Language}}},
		},
	}
}

func (lookup *Lookup) searchResponseTo(
	ctx context.Context,
	body []byte,
) (searchResponse, error) {
	target := lookup.endpoint + "/" + lookup.indexes.Prefix() + "_*/_search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return searchResponse{}, fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := lookup.client.Do(req)
	if err != nil {
		return searchResponse{}, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusMultipleChoices {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, detailLimit))
		return searchResponse{}, fmt.Errorf("search: status %d: %s", resp.StatusCode, detail)
	}
	var response searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return searchResponse{}, fmt.Errorf("decode search response: %w", err)
	}
	return response, nil
}

type searchResponse struct {
	Hits struct {
		Hits []searchHit `json:"hits"`
	} `json:"hits"`
}

type searchHit struct {
	Source struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"_source"`
	Highlight struct {
		Content []string `json:"content"`
	} `json:"highlight"`
}

func (response searchResponse) pageMatches() []searchdocument.PageMatch {
	matches := make([]searchdocument.PageMatch, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		matches = append(matches, searchdocument.PageMatch{
			URL:     hit.Source.URL,
			Title:   hit.Source.Title,
			Snippet: strings.Join(hit.Highlight.Content, " "),
		})
	}
	return matches
}
