// Package spamlast orders the found documents as another ordering does, and
// then puts each document whose page the spam assessment calls spam after all
// the other documents. The documents on each side of that line keep the order
// of the other ordering.
package spamlast

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

type DocumentsOrdering interface {
	OrderedDocumentsOf(findings queryfindings.Findings) []queryfindings.FoundDocument
}

type Ordering struct {
	documentsOrdering DocumentsOrdering
}

func New(documentsOrdering DocumentsOrdering) Ordering {
	return Ordering{documentsOrdering: documentsOrdering}
}

func (ordering Ordering) OrderedDocumentsOf(
	findings queryfindings.Findings,
) []queryfindings.FoundDocument {
	return documentsWithSpamLast(ordering.documentsOrdering.OrderedDocumentsOf(findings))
}

func documentsWithSpamLast(
	orderedDocuments []queryfindings.FoundDocument,
) []queryfindings.FoundDocument {
	otherDocuments := make([]queryfindings.FoundDocument, 0, len(orderedDocuments))
	var spamDocuments []queryfindings.FoundDocument
	for _, orderedDocument := range orderedDocuments {
		if orderedDocument.IsSpam() {
			spamDocuments = append(spamDocuments, orderedDocument)
			continue
		}
		otherDocuments = append(otherDocuments, orderedDocument)
	}

	return append(otherDocuments, spamDocuments...)
}
