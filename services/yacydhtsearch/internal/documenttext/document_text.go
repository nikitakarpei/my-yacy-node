// Package documenttext derives the text of a document from its page: the
// title of the page, how often the text holds each query word and each query
// phrase, how many words it holds, and the snippet, which is the run of
// sentences that answers the query best. A query phrase is two words the query
// puts side by side.
package documenttext

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentText struct {
	Title            string
	HitsPerQueryWord map[yacymodel.Hash]int
	QueryPhraseHits  int
	AmountOfWords    int
	Snippet          string
}

func DocumentTextFrom(
	pageTitle string,
	pageText string,
	queryWords []yacymodel.Hash,
	snippetLengthCeiling int,
) DocumentText {
	counts := textCountsOf(pageText, queryWords)

	return DocumentText{
		Title:            strings.Join(strings.Fields(pageTitle), " "),
		HitsPerQueryWord: counts.hitsPerQueryWord,
		QueryPhraseHits:  counts.queryPhraseHits,
		AmountOfWords:    counts.amountOfWords,
		Snippet:          snippetOf(pageText, queryWords, snippetLengthCeiling),
	}
}
