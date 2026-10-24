// Package textsnippet chooses the snippet of a text for a query: the run of whole
// sentences, within a length ceiling, that holds the most distinct query words, then
// the most query phrases, then the most query words. A sentence longer than the
// ceiling is cut at spaces around its first query word. The service names how its
// text splits into words, so query words match as its search matches them.
package textsnippet

import (
	"iter"
	"strings"
)

type Rule struct {
	WordsIn       func(text string) iter.Seq2[int, string]
	LengthCeiling int
}

func (rule Rule) SnippetOf(text string, queryWords []string) string {
	query := queryOf(queryWords)
	sentences := sentencesIn(text, rule.WordsIn)
	bestPassage := query.bestPassageIn(sentences, rule.LengthCeiling)
	return withSpacesCollapsed(bestPassage.textWithin(text, query, rule.LengthCeiling))
}

func withSpacesCollapsed(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
