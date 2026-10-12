// Package pagecontents derives the contents of the page of a document: the
// address the page moved to, if it moved, the title of the page, how often the
// text and the auxiliary text hold each query word, how often the title, the
// text and the auxiliary text hold each query phrase, how many words the text
// holds, the snippet, which is the run of sentences of the text that answers the
// query best, and how many links of its own site and of other sites it holds. A
// query phrase is two words the query puts side by side, in either order.
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

type ExtractedPage struct {
	Title         string
	Text          string
	AuxiliaryText string
	LinkCounts    LinkCounts
}

func PageContentsFrom(
	extractedPage ExtractedPage,
	queryWords []string,
	snippetLengthCeiling int,
) PageContents {
	counts := textCountsOf(extractedPage.Text, queryWords)
	auxiliaryTextCounts := textCountsOf(extractedPage.AuxiliaryText, queryWords)
	phrases := queryPhrasesOf(queryWords)

	return PageContents{
		Title:            strings.Join(strings.Fields(extractedPage.Title), " "),
		HitsPerQueryWord: counts.hitsPerWordHashWith(auxiliaryTextCounts),
		QueryPhraseHits: phrases.hitsIn(extractedPage.Title) + phrases.hitsIn(extractedPage.Text) +
			phrases.hitsIn(extractedPage.AuxiliaryText),
		AmountOfWords: counts.amountOfWords,
		Snippet:       snippetOf(extractedPage.Text, queryWords, snippetLengthCeiling),
		LinkCounts:    extractedPage.LinkCounts,
	}
}
