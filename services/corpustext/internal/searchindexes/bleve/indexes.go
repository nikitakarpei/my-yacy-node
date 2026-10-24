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
	"github.com/blevesearch/bleve/v2/analysis"
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
	blevematch "github.com/blevesearch/bleve/v2/search"
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
	analyzers       map[string]analysis.Analyzer
	everyIndex      blevesearch.IndexAlias
}

func New(directory string, languageIndexes languageindex.LanguageIndexes) *Indexes {
	return &Indexes{directory: directory, languageIndexes: languageIndexes}
}

func (indexes *Indexes) Open(ctx context.Context) error {
	if err := os.MkdirAll(indexes.directory, directoryPermission); err != nil {
		return fmt.Errorf("create search index directory %s: %w", indexes.directory, err)
	}
	openIndexes := make(map[string]blevesearch.Index, len(indexes.languageIndexes.All()))
	analyzers := make(map[string]analysis.Analyzer, len(indexes.languageIndexes.All()))
	for _, languageIndex := range indexes.languageIndexes.All() {
		path := filepath.Join(indexes.directory, languageIndex.Name)
		if err := createIfAbsent(ctx, path, languageIndex); err != nil {
			return errors.Join(err, closeAll(openIndexes))
		}
		openIndex, err := blevesearch.Open(path)
		if err != nil {
			return errors.Join(
				fmt.Errorf("open index %s: %w", languageIndex.Name, err),
				closeAll(openIndexes),
			)
		}
		openIndexes[languageIndex.Name] = openIndex
		analyzers[languageIndex.Name] = analyzerIn(openIndex, languageIndex.Language)
	}
	indexes.openIndexes = openIndexes
	indexes.analyzers = analyzers
	indexes.everyIndex = blevesearch.NewIndexAlias(slices.Collect(maps.Values(openIndexes))...)
	return nil
}

func createIfAbsent(
	ctx context.Context,
	path string,
	languageIndex languageindex.LanguageIndex,
) error {
	if _, err := os.Stat(path); err == nil {
		slog.DebugContext(ctx, "bleve index present", slog.String("index", languageIndex.Name))
		return nil
	}
	indexMapping, err := indexMappingFor(languageIndex.Language)
	if err != nil {
		return fmt.Errorf("map index %s: %w", languageIndex.Name, err)
	}
	created, err := blevesearch.New(path, indexMapping)
	if err != nil {
		return fmt.Errorf("create index %s: %w", languageIndex.Name, err)
	}
	if err := created.Close(); err != nil {
		return fmt.Errorf("close created index %s: %w", languageIndex.Name, err)
	}
	slog.DebugContext(ctx, "bleve index created", slog.String("index", languageIndex.Name))
	return nil
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
	indexMapping.DefaultAnalyzer = analyzerNameOf(language)
	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false
	indexMapping.DocValuesDynamic = false
	indexMapping.DefaultMapping = documentMapping()
	return indexMapping, nil
}

func analyzerIn(openIndex blevesearch.Index, language string) analysis.Analyzer {
	return openIndex.Mapping().AnalyzerNamed(analyzerNameOf(language))
}

func analyzerNameOf(language string) string {
	analyzer, defined := languageAnalyzers[language]
	if !defined {
		return undeterminedAnalyzer
	}
	return analyzer
}

func documentMapping() *mapping.DocumentMapping {
	title := textField()
	title.Store = true
	content := textField()
	content.Store = true
	url := keywordField()
	url.Store = true
	url.DocValues = true
	language := keywordField()
	language.Store = true

	document := blevesearch.NewDocumentStaticMapping()
	document.AddFieldMappingsAt("title", title)
	document.AddFieldMappingsAt("content", content)
	document.AddFieldMappingsAt("auxiliary_text", textField())
	document.AddFieldMappingsAt("url", url)
	document.AddFieldMappingsAt("language", language)
	return document
}

func textField() *mapping.FieldMapping {
	field := blevesearch.NewTextFieldMapping()
	field.Store = false
	field.IncludeTermVectors = false
	field.DocValues = false
	return field
}

func keywordField() *mapping.FieldMapping {
	field := blevesearch.NewKeywordFieldMapping()
	field.Analyzer = keyword.Name
	field.Store = false
	field.IncludeTermVectors = false
	field.IncludeInAll = false
	field.DocValues = false
	return field
}

func closeAll(openIndexes map[string]blevesearch.Index) error {
	var failures error
	for name, opened := range openIndexes {
		if err := opened.Close(); err != nil {
			failures = errors.Join(failures, fmt.Errorf("close index %s: %w", name, err))
		}
	}
	return failures
}

func (indexes *Indexes) Index(_ context.Context, document searchdocument.Document) error {
	openIndex := indexes.openIndexes[indexes.languageIndexes.NameFor(document.Language)]
	if err := openIndex.Index(document.URL, document); err != nil {
		return fmt.Errorf("index document %s: %w", document.URL, err)
	}
	return nil
}

func (indexes *Indexes) MatchesFor(
	ctx context.Context,
	search textsearch.Search,
) ([]searchdocument.PageMatch, error) {
	rankedURLs, err := indexes.rankedURLsFor(ctx, search)
	if err != nil {
		return nil, err
	}
	return indexes.pageMatchesOf(ctx, search, rankedURLs)
}

func (indexes *Indexes) rankedURLsFor(
	ctx context.Context,
	search textsearch.Search,
) ([]string, error) {
	request := blevesearch.NewSearchRequestOptions(
		queryFor(search), search.Limit, search.Offset, false,
	)
	request.SortBy([]string{"-_score", "url"})
	result, err := indexes.everyIndex.SearchInContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("rank matches: %w", err)
	}
	rankedURLs := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		rankedURLs = append(rankedURLs, hit.ID)
	}
	return rankedURLs, nil
}

func (indexes *Indexes) pageMatchesOf(
	ctx context.Context,
	search textsearch.Search,
	rankedURLs []string,
) ([]searchdocument.PageMatch, error) {
	request := blevesearch.NewSearchRequestOptions(
		blevesearch.NewDocIDQuery(rankedURLs), len(rankedURLs), 0, false,
	)
	request.Fields = []string{"title", "url", "content", "language"}
	result, err := indexes.everyIndex.SearchInContext(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("load matches: %w", err)
	}
	return indexes.pageMatchesIn(search, rankedURLs, result.Hits), nil
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

func weightedMatchFor(searchQuery, field string, weight int) query.Query {
	match := blevesearch.NewMatchQuery(searchQuery)
	match.SetField(field)
	match.SetBoost(float64(weight))
	return match
}

func (indexes *Indexes) pageMatchesIn(
	search textsearch.Search,
	rankedURLs []string,
	hits blevematch.DocumentMatchCollection,
) []searchdocument.PageMatch {
	slices.SortFunc(hits, func(first, second *blevematch.DocumentMatch) int {
		return slices.Index(rankedURLs, first.ID) - slices.Index(rankedURLs, second.ID)
	})
	matches := make([]searchdocument.PageMatch, 0, len(hits))
	for _, hit := range hits {
		matches = append(matches, searchdocument.PageMatch{
			URL:     textOf(hit, "url"),
			Title:   textOf(hit, "title"),
			Snippet: indexes.snippetOf(hit, search.Query),
		})
	}
	return matches
}

func (indexes *Indexes) snippetOf(hit *blevematch.DocumentMatch, searchQuery string) string {
	analyzer := indexes.analyzers[indexes.languageIndexes.NameFor(textOf(hit, "language"))]
	return snippetFrom(textOf(hit, "content"), searchQuery, analyzer)
}

func textOf(hit *blevematch.DocumentMatch, field string) string {
	text, _ := hit.Fields[field].(string)
	return text
}

func (indexes *Indexes) Close(ctx context.Context) {
	if err := closeAll(indexes.openIndexes); err != nil {
		slog.WarnContext(ctx, "bleve index close failed", slog.Any("error", err))
	}
}
