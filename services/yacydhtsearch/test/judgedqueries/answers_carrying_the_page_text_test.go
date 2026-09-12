package judgedqueries_test

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answersCarryingThePageTextOfEachDocument(
	query string,
	answers peeranswers.AnsweredQuery,
	pageTextPerDocument map[yacymodel.URLHash]string,
) peeranswers.AnsweredQuery {
	queryWords := searchquery.QueryFrom(query, "").TermHashes()

	documentTextPerDocument := make(
		map[yacymodel.URLHash]documenttext.DocumentText, len(pageTextPerDocument),
	)
	for document, pageText := range pageTextPerDocument {
		documentTextPerDocument[document] = documenttext.DocumentTextFrom(
			pageText, queryWords, snippetLengthCeiling,
		)
	}

	return answers.CarryingTheTextOfEachDocument(documentTextPerDocument)
}
