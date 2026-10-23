package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/hitsreply"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

type Lookup struct {
	target string
	client *http.Client
}

func NewLookup(
	endpoint string,
	indexes languageindex.LanguageIndexes,
	client *http.Client,
) *Lookup {
	return &Lookup{
		target: strings.TrimRight(endpoint, "/") + "/" + indexes.Prefix() + "_*/_search",
		client: client,
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
	return hitsreply.PageMatchesFor(ctx, lookup.client, lookup.target, body)
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
