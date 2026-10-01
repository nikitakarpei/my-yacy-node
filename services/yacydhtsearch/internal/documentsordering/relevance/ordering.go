// Package relevance orders the found documents by how well each document
// findings the query, the most relevant document first. Documents of equal
// relevance keep the order the spread found them in.
package relevance

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentRelevance interface {
	RelevancePerDocumentOf(findings queryfindings.Findings) map[yacymodel.URLHash]float64
}

type Ordering struct {
	documentRelevance DocumentRelevance
}

func New(documentRelevance DocumentRelevance) Ordering {
	return Ordering{documentRelevance: documentRelevance}
}

func (ordering Ordering) OrderedDocumentsOf(
	findings queryfindings.Findings,
) []queryfindings.FoundDocument {
	return documentsInFallingOrderOfRelevance(
		slices.Clone(findings.FoundDocuments),
		ordering.documentRelevance.RelevancePerDocumentOf(findings),
	)
}

func documentsInFallingOrderOfRelevance(
	foundDocuments []queryfindings.FoundDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []queryfindings.FoundDocument {
	slices.SortStableFunc(foundDocuments, func(one, other queryfindings.FoundDocument) int {
		return cmp.Compare(
			relevancePerDocument[other.Hash], relevancePerDocument[one.Hash],
		)
	})

	return foundDocuments
}
