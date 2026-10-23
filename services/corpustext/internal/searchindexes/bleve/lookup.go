package bleve

import (
	"context"
	"fmt"

	blevesearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

const everyField = "_all"

type Lookup struct {
	indexes *Indexes
}

func NewLookup(indexes *Indexes) *Lookup {
	return &Lookup{indexes: indexes}
}

func (lookup *Lookup) MatchesFor(
	ctx context.Context,
	search textsearch.Search,
) ([]searchdocument.PageMatch, error) {
	result, err := lookup.indexes.everyIndex().SearchInContext(ctx, searchRequestFor(search))
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return pageMatchesIn(result.Hits), nil
}

func searchRequestFor(search textsearch.Search) *blevesearch.SearchRequest {
	request := blevesearch.NewSearchRequestOptions(
		queryFor(search), search.Limit, search.Offset, false,
	)
	request.Fields = []string{"title", "url"}
	request.Highlight = blevesearch.NewHighlightWithStyle(snippetHighlighter)
	request.Highlight.AddField("content")
	return request
}

func queryFor(search textsearch.Search) query.Query {
	everyWord := blevesearch.NewMatchQuery(search.Query)
	everyWord.SetField(everyField)
	everyWord.Operator = query.MatchQueryOperatorAnd
	ranking := blevesearch.NewBooleanQuery()
	ranking.AddMust(everyWord)
	ranking.AddShould(
		weightedMatchFor(search.Query, "title", textsearch.TitleWeight),
		weightedMatchFor(search.Query, "content", textsearch.ContentWeight),
		weightedMatchFor(search.Query, "auxiliary_text", textsearch.AuxiliaryTextWeight),
	)
	if search.Language != "" {
		language := blevesearch.NewTermQuery(search.Language)
		language.SetField("language")
		ranking.AddMust(language)
	}
	return ranking
}

func weightedMatchFor(words, field string, weight int) query.Query {
	match := blevesearch.NewMatchQuery(words)
	match.SetField(field)
	match.SetBoost(float64(weight))
	return match
}

func pageMatchesIn(hits search.DocumentMatchCollection) []searchdocument.PageMatch {
	matches := make([]searchdocument.PageMatch, 0, len(hits))
	for _, hit := range hits {
		matches = append(matches, searchdocument.PageMatch{
			URL:     textOf(hit, "url"),
			Title:   textOf(hit, "title"),
			Snippet: snippetOf(hit),
		})
	}
	return matches
}

func textOf(hit *search.DocumentMatch, field string) string {
	text, _ := hit.Fields[field].(string)
	return text
}

func snippetOf(hit *search.DocumentMatch) string {
	fragments := hit.Fragments["content"]
	if len(fragments) == 0 {
		return ""
	}
	return fragments[0]
}
