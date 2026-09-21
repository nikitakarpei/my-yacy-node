package judgedqueries_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type answersAndTheirReadPages struct {
	answeredQuery           queryanswers.AnsweredQuery
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents
}

func (e pageExtraction) answersWithTheContentsOfTheStoredPages(
	ctx context.Context,
	t *testing.T,
	query string,
	answers queryanswers.AnsweredQuery,
	pagePerAddress map[string]storedPage,
) answersAndTheirReadPages {
	t.Helper()

	queryWords := searchquery.QueryFrom(query, "").TermHashes()
	pageContentsPerDocument := map[yacymodel.URLHash]pagecontents.PageContents{}
	for _, foundDocument := range answers.FoundDocuments {
		page, stored := pagePerAddress[foundDocument.Address]
		if !stored {
			continue
		}
		extracted := e.extractedPageOf(ctx, page)
		if extracted.text == "" {
			continue
		}
		pageContentsPerDocument[foundDocument.Hash] = pagecontents.PageContentsFrom(
			extracted.title,
			extracted.text,
			extracted.linkCounts,
			queryWords,
			snippetLengthCeiling,
		)
	}

	return answersAndTheirReadPages{
		answeredQuery:           answers,
		pageContentsPerDocument: pageContentsPerDocument,
	}
}
