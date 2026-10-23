package bleve

import (
	"fmt"

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
)

const undeterminedAnalyzer = "corpus_text_und"

var languageAnalyzers = map[string]string{
	"en": en.AnalyzerName,
	"de": de.AnalyzerName,
	"fr": fr.AnalyzerName,
	"ru": ru.AnalyzerName,
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
	document := blevesearch.NewDocumentStaticMapping()
	document.AddFieldMappingsAt("title", textField(true))
	document.AddFieldMappingsAt("content", textField(true))
	document.AddFieldMappingsAt("auxiliary_text", textField(false))
	document.AddFieldMappingsAt("url", keywordField(true))
	document.AddFieldMappingsAt("language", keywordField(false))
	crawledAt := blevesearch.NewDateTimeFieldMapping()
	crawledAt.IncludeInAll = false
	document.AddFieldMappingsAt("crawled_at", crawledAt)
	return document
}

func textField(stored bool) *mapping.FieldMapping {
	field := blevesearch.NewTextFieldMapping()
	field.Store = stored
	field.IncludeTermVectors = stored
	field.DocValues = false
	return field
}

func keywordField(stored bool) *mapping.FieldMapping {
	field := blevesearch.NewKeywordFieldMapping()
	field.Analyzer = keyword.Name
	field.Store = stored
	field.IncludeInAll = false
	field.DocValues = false
	return field
}
