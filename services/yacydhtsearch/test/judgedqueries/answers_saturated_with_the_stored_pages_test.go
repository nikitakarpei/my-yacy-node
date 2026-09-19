package judgedqueries_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type saturatedAnswers struct {
	answers                 queryanswers.AnsweredQuery
	pageContentsPerDocument map[yacymodel.URLHash]queryanswers.PageContents
}

func (e pageExtraction) answersSaturatedWithTheStoredPages(
	ctx context.Context,
	t *testing.T,
	query string,
	answers queryanswers.AnsweredQuery,
	pagePerAddress map[string]storedPage,
) saturatedAnswers {
	t.Helper()

	queryWords := searchquery.QueryFrom(query, "").TermHashes()
	pageContentsPerDocument := map[yacymodel.URLHash]queryanswers.PageContents{}
	for _, foundDocument := range answers.FoundDocuments {
		page, stored := pagePerAddress[foundDocument.Address]
		if !stored {
			continue
		}
		extracted := e.extractedPageOf(ctx, page)
		if extracted.text == "" {
			continue
		}
		pageContentsPerDocument[foundDocument.Hash] = queryanswers.PageContents{
			Text: documenttext.DocumentTextFrom(
				extracted.title, extracted.text, queryWords, snippetLengthCeiling,
			),
			LinkCounts: extracted.linkCounts,
		}
	}

	return saturatedAnswers{
		answers:                 answers.SaturatedWith(pageContentsPerDocument),
		pageContentsPerDocument: pageContentsPerDocument,
	}
}
