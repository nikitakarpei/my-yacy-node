package bleve

import (
	"unicode/utf8"

	"github.com/blevesearch/bleve/v2/analysis"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

func snippetFrom(content, searchQuery string, analyzer analysis.Analyzer) string {
	runes := []rune(content)
	end := min(len(runes), firstMatchIn(content, searchQuery, analyzer)+textsearch.SnippetLength/2)
	end = min(len(runes), max(end, textsearch.SnippetLength))
	return string(runes[max(0, end-textsearch.SnippetLength):end])
}

func firstMatchIn(content, searchQuery string, analyzer analysis.Analyzer) int {
	queryTerms := make(map[string]struct{})
	for _, token := range analyzer.Analyze([]byte(searchQuery)) {
		queryTerms[string(token.Term)] = struct{}{}
	}
	for _, token := range analyzer.Analyze([]byte(content)) {
		if _, matches := queryTerms[string(token.Term)]; matches {
			return utf8.RuneCountInString(content[:token.Start])
		}
	}
	return 0
}
