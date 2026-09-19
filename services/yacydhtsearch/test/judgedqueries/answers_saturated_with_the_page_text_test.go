package judgedqueries_test

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answersSaturatedWithThePageText(
	query string,
	answers queryanswers.AnsweredQuery,
	pageTextPerDocument map[yacymodel.URLHash]string,
	pageTitlePerDocument map[yacymodel.URLHash]string,
) queryanswers.AnsweredQuery {
	queryWords := searchquery.QueryFrom(query, "").TermHashes()

	documentTextPerDocument := make(
		map[yacymodel.URLHash]documenttext.DocumentText, len(pageTextPerDocument),
	)
	for document, pageText := range pageTextPerDocument {
		documentTextPerDocument[document] = documenttext.DocumentTextFrom(
			pageTitlePerDocument[document], pageText, queryWords, snippetLengthCeiling,
		)
	}

	return answers.SaturatedWith(documentTextPerDocument)
}
