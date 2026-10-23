package manticore

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

const wordsAroundAMatch = 25

type Lookup struct {
	endpoint string
	tables   languageindex.LanguageIndexes
	client   *http.Client
}

func NewLookup(
	endpoint string,
	tables languageindex.LanguageIndexes,
	client *http.Client,
) *Lookup {
	return &Lookup{
		endpoint: strings.TrimRight(endpoint, "/"),
		tables:   tables,
		client:   client,
	}
}

func (lookup *Lookup) MatchesFor(
	ctx context.Context,
	search textsearch.Search,
) ([]searchdocument.PageMatch, error) {
	body, err := json.Marshal(lookup.searchRequestFor(search))
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
	Table     string         `json:"table"`
	Offset    int            `json:"offset"`
	Limit     int            `json:"limit"`
	Query     map[string]any `json:"query"`
	Options   map[string]any `json:"options"`
	Highlight map[string]any `json:"highlight"`
	Source    []string       `json:"_source"`
}

func (lookup *Lookup) searchRequestFor(search textsearch.Search) searchRequest {
	return searchRequest{
		Table:  lookup.tables.Prefix(),
		Offset: search.Offset,
		Limit:  search.Limit,
		Query:  queryFor(search),
		Options: map[string]any{
			"field_weights": map[string]int{
				"title":          textsearch.TitleWeight,
				"content":        textsearch.ContentWeight,
				"auxiliary_text": textsearch.AuxiliaryTextWeight,
			},
		},
		Highlight: map[string]any{
			"fields":           []string{"content"},
			"before_match":     "",
			"after_match":      "",
			"limit":            textsearch.SnippetLength,
			"around":           wordsAroundAMatch,
			"passage_boundary": "sentence",
			"limit_passages":   1,
			"weight_order":     true,
		},
		Source: []string{"title", "url"},
	}
}

func queryFor(search textsearch.Search) map[string]any {
	everyWord := map[string]any{
		"match": map[string]any{
			"title,content,auxiliary_text": map[string]any{
				"query":    search.Query,
				"operator": "and",
			},
		},
	}
	if search.Language == "" {
		return everyWord
	}
	return map[string]any{
		"bool": map[string]any{
			"must": []any{
				everyWord,
				map[string]any{"equals": map[string]any{"language": search.Language}},
			},
		},
	}
}

func (lookup *Lookup) searchResponseTo(
	ctx context.Context,
	body []byte,
) (searchResponse, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		lookup.endpoint+"/search",
		bytes.NewReader(body),
	)
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
