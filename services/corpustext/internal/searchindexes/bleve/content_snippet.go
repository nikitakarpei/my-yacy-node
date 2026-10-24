package bleve

import (
	"iter"

	"github.com/blevesearch/bleve/v2/analysis"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/textsnippet"
)

func snippetFrom(content, searchQuery string, analyzer analysis.Analyzer) string {
	rule := textsnippet.Rule{
		WordsIn:       placedTermsFor(analyzer),
		LengthCeiling: textsearch.SnippetLength,
	}
	return rule.SnippetOf(content, termsOf(searchQuery, analyzer))
}

func placedTermsFor(analyzer analysis.Analyzer) func(text string) iter.Seq2[int, string] {
	return func(text string) iter.Seq2[int, string] {
		return func(yield func(int, string) bool) {
			for _, token := range analyzer.Analyze([]byte(text)) {
				if !yield(token.Start, string(token.Term)) {
					return
				}
			}
		}
	}
}

func termsOf(searchQuery string, analyzer analysis.Analyzer) []string {
	var terms []string
	for _, term := range placedTermsFor(analyzer)(searchQuery) {
		terms = append(terms, term)
	}
	return terms
}
