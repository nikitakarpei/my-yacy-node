// Package relevance orders the found documents by how well each document
// answers the query, the most relevant document first. Documents of equal
// relevance keep the order the spread found them in.
package relevance

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentRelevance interface {
	RelevancePerDocumentOf(answers queryanswers.AnsweredQuery) map[yacymodel.URLHash]float64
}

type Ordering struct {
	documentRelevance DocumentRelevance
}

func New(documentRelevance DocumentRelevance) Ordering {
	return Ordering{documentRelevance: documentRelevance}
}

func (ordering Ordering) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return documentsInFallingOrderOfRelevance(
		slices.Clone(answers.FoundDocuments),
		ordering.documentRelevance.RelevancePerDocumentOf(answers),
	)
}

func documentsInFallingOrderOfRelevance(
	foundDocuments []queryanswers.FoundDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []queryanswers.FoundDocument {
	slices.SortStableFunc(foundDocuments, func(one, other queryanswers.FoundDocument) int {
		return cmp.Compare(
			relevancePerDocument[other.Hash], relevancePerDocument[one.Hash],
		)
	})

	return foundDocuments
}
