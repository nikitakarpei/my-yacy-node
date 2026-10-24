package bleve

import (
	"iter"

	"github.com/blevesearch/bleve/v2/analysis"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/textsnippet"
)

func snippetFrom(title, content, searchQuery string, analyzer analysis.Analyzer) string {
	rule := textsnippet.Rule{
		WordsIn:       wordsAnalyzedBy(analyzer),
		LengthCeiling: textsearch.SnippetLength,
	}
	return rule.SnippetOf(title, content, queryWordsOf(searchQuery, analyzer))
}

func wordsAnalyzedBy(analyzer analysis.Analyzer) func(text string) iter.Seq2[int, string] {
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

func queryWordsOf(searchQuery string, analyzer analysis.Analyzer) []string {
	var queryWords []string
	for _, queryWord := range wordsAnalyzedBy(analyzer)(searchQuery) {
		queryWords = append(queryWords, queryWord)
	}
	return queryWords
}
