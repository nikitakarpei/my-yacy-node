// Package bleve keeps the search index of the text corpus in Bleve indexes in a directory
// of the local filesystem: one index per configured language and one for all other pages.
// It writes documents into them and answers searches of all of them at once.
package bleve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"

	blevesearch "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/char/asciifolding"
	"github.com/blevesearch/bleve/v2/analysis/lang/de"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/lang/fr"
	"github.com/blevesearch/bleve/v2/analysis/lang/ru"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

const (
	directoryPermission  = 0o750
	undeterminedAnalyzer = "corpus_text_und"
	everyField           = "_all"
)

var languageAnalyzers = map[string]string{
	"en": en.AnalyzerName,
	"de": de.AnalyzerName,
	"fr": fr.AnalyzerName,
	"ru": ru.AnalyzerName,
}

type Indexes struct {
	directory       string
	languageIndexes languageindex.LanguageIndexes
	openIndexes     map[string]blevesearch.Index
	alias           blevesearch.IndexAlias
}

func NewIndexes(directory string, languageIndexes languageindex.LanguageIndexes) *Indexes {
	return &Indexes{directory: directory, languageIndexes: languageIndexes}
}

func (indexes *Indexes) Bootstrap(ctx context.Context) error {
	if err := os.MkdirAll(indexes.directory, directoryPermission); err != nil {
		return fmt.Errorf("create search index directory %s: %w", indexes.directory, err)
	}
	openIndexes := make(map[string]blevesearch.Index, len(indexes.languageIndexes.All()))
	for _, languageIndex := range indexes.languageIndexes.All() {
		opened, err := indexes.openedOrCreated(ctx, languageIndex)
		if err != nil {
			return errors.Join(err, closed(openIndexes))
		}
		openIndexes[languageIndex.Name] = opened
	}
	indexes.openIndexes = openIndexes
	indexes.alias = blevesearch.NewIndexAlias(slices.Collect(maps.Values(openIndexes))...)
	return nil
}

func (indexes *Indexes) openedOrCreated(
	ctx context.Context,
	languageIndex languageindex.LanguageIndex,
) (blevesearch.Index, error) {
	path := filepath.Join(indexes.directory, languageIndex.Name)
	opened, err := blevesearch.Open(path)
	if err == nil {
		slog.DebugContext(ctx, "bleve index present", slog.String("index", languageIndex.Name))
		return opened, nil
	}
	if !errors.Is(err, blevesearch.ErrorIndexPathDoesNotExist) {
		return nil, fmt.Errorf("open index %s: %w", languageIndex.Name, err)
	}
	indexMapping, err := indexMappingFor(languageIndex.Language)
	if err != nil {
		return nil, fmt.Errorf("map index %s: %w", languageIndex.Name, err)
	}
	created, err := blevesearch.New(path, indexMapping)
	if err != nil {
		return nil, fmt.Errorf("create index %s: %w", languageIndex.Name, err)
	}
	slog.DebugContext(ctx, "bleve index created", slog.String("index", languageIndex.Name))
	return created, nil
}

func indexMappingFor(language string) (*mapping.IndexMappingImpl, error) {
	indexMapping := blevesearch.NewIndexMapping()
	if err := indexMapping.AddCustomAnalyzer(undeterminedAnalyzer, map[string]any{
		"type":          custom.Name,
		"char_filters":  []any{asciifolding.Name},
		"tokenizer":     unicode.Name,
		"token_filters": []any{lowercase.Name},
	}); err != nil {
		return nil, fmt.Errorf("define analyzer %s: %w", undeterminedAnalyzer, err)
	}
	indexMapping.DefaultAnalyzer = analyzerOf(language)
	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false
	indexMapping.DocValuesDynamic = false
	indexMapping.DefaultMapping = documentMapping()
	return indexMapping, nil
}

func analyzerOf(language string) string {
	analyzer, defined := languageAnalyzers[language]
	if !defined {
		return undeterminedAnalyzer
	}
	return analyzer
}

func documentMapping() *mapping.DocumentMapping {
	title := searchedTextField()
	title.Store = true
	content := searchedTextField()
	content.Store = true
	content.IncludeTermVectors = true
	url := searchedKeywordField()
	url.Index = false
	url.Store = true

	document := blevesearch.NewDocumentStaticMapping()
	document.AddFieldMappingsAt("title", title)
	document.AddFieldMappingsAt("content", content)
	document.AddFieldMappingsAt("auxiliary_text", searchedTextField())
	document.AddFieldMappingsAt("url", url)
	document.AddFieldMappingsAt("language", searchedKeywordField())
	return document
}

func searchedTextField() *mapping.FieldMapping {
	field := blevesearch.NewTextFieldMapping()
	field.Store = false
	field.IncludeTermVectors = false
	field.DocValues = false
	return field
}

func searchedKeywordField() *mapping.FieldMapping {
	field := blevesearch.NewKeywordFieldMapping()
	field.Analyzer = keyword.Name
	field.Store = false
	field.IncludeTermVectors = false
	field.IncludeInAll = false
	field.DocValues = false
	return field
}

func closed(openIndexes map[string]blevesearch.Index) error {
	var failures error
	for name, opened := range openIndexes {
		if err := opened.Close(); err != nil {
			failures = errors.Join(failures, fmt.Errorf("close index %s: %w", name, err))
		}
	}
	return failures
}

func (indexes *Indexes) Index(_ context.Context, document searchdocument.Document) error {
	languageIndex := indexes.openIndexes[indexes.languageIndexes.NameFor(document.Language)]
	if err := languageIndex.Index(document.URL, document); err != nil {
		return fmt.Errorf("index document %s: %w", document.URL, err)
	}
	return nil
}

func (indexes *Indexes) MatchesFor(
	ctx context.Context,
	search textsearch.Search,
) ([]searchdocument.PageMatch, error) {
	result, err := indexes.alias.SearchInContext(ctx, searchRequestFor(search))
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

func (indexes *Indexes) Close(ctx context.Context) {
	if err := closed(indexes.openIndexes); err != nil {
		slog.WarnContext(ctx, "bleve index close failed", slog.Any("error", err))
	}
}
