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
	documentTextPerDocument map[yacymodel.URLHash]documenttext.DocumentText
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
	documentTextPerDocument := map[yacymodel.URLHash]documenttext.DocumentText{}
	linkCountsPerDocument := map[yacymodel.URLHash]queryanswers.LinkCounts{}
	for _, foundDocument := range answers.FoundDocuments {
		page, stored := pagePerAddress[foundDocument.Address]
		if !stored {
			continue
		}
		extracted := e.extractedPageOf(ctx, page)
		if extracted.text == "" {
			continue
		}
		documentTextPerDocument[foundDocument.Hash] = documenttext.DocumentTextFrom(
			extracted.title, extracted.text, queryWords, snippetLengthCeiling,
		)
		linkCountsPerDocument[foundDocument.Hash] = extracted.linkCounts
	}

	return saturatedAnswers{
		answers: answersCarryingTheLinkCounts(
			answers.SaturatedWith(documentTextPerDocument), linkCountsPerDocument,
		),
		documentTextPerDocument: documentTextPerDocument,
	}
}

func answersCarryingTheLinkCounts(
	answers queryanswers.AnsweredQuery,
	linkCountsPerDocument map[yacymodel.URLHash]queryanswers.LinkCounts,
) queryanswers.AnsweredQuery {
	foundDocuments := make([]queryanswers.FoundDocument, 0, len(answers.FoundDocuments))
	for _, foundDocument := range answers.FoundDocuments {
		linkCounts, extracted := linkCountsPerDocument[foundDocument.Hash]
		if extracted {
			foundDocument.LinkCounts = yacymodel.Some(linkCounts)
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	answers.FoundDocuments = foundDocuments

	return answers
}
