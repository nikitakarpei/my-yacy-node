// Package pagecontents derives the contents of the page of a document: the
// address the page moved to, if it moved, the title of the page, how often the
// text holds each query word, how often the title and the text hold each query
// phrase, how many words the text holds, the snippet, which is the run of
// sentences that answers the query best, and how many links of its own site and
// of other sites it holds. A query phrase is two words the query puts side by
// side, in either order.
package pagecontents

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageContents struct {
	Address          string
	Title            string
	HitsPerQueryWord map[yacymodel.Hash]int
	QueryPhraseHits  int
	AmountOfWords    int
	Snippet          string
	LinkCounts       LinkCounts
}

func PageContentsFrom(
	pageTitle string,
	pageText string,
	linkCounts LinkCounts,
	queryWords []yacymodel.Hash,
	snippetLengthCeiling int,
) PageContents {
	counts := textCountsOf(pageText, queryWords)
	phrases := queryPhrasesOf(queryWords)

	return PageContents{
		Title:            strings.Join(strings.Fields(pageTitle), " "),
		HitsPerQueryWord: counts.hitsPerQueryWord,
		QueryPhraseHits:  phrases.hitsIn(pageTitle) + phrases.hitsIn(pageText),
		AmountOfWords:    counts.amountOfWords,
		Snippet:          snippetOf(pageText, queryWords, snippetLengthCeiling),
		LinkCounts:       linkCounts,
	}
}
