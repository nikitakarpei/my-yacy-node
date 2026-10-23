package manticore

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

const wordsAroundAMatch = 25

type Lookup struct {
	target string
	table  string
	client *http.Client
}

func NewLookup(
	endpoint string,
	tables languageindex.LanguageIndexes,
	client *http.Client,
) *Lookup {
	return &Lookup{
		target: strings.TrimRight(endpoint, "/") + "/search",
		table:  tables.Prefix(),
		client: client,
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
	return hitsreply.PageMatchesFor(ctx, lookup.client, lookup.target, body)
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
		Table:  lookup.table,
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
