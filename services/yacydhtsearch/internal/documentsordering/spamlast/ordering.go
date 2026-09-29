// Package spamlast orders the found documents as another ordering does, and
// then puts each document whose page the spam assessment calls spam after all
// the other documents. The documents on each side of that line keep the order
// of the other ordering.
package spamlast

import (
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

type DocumentsOrdering interface {
	OrderedDocumentsOf(answers queryanswers.AnsweredQuery) []queryanswers.FoundDocument
}

type Ordering struct {
	documentsOrdering DocumentsOrdering
}

func New(documentsOrdering DocumentsOrdering) Ordering {
	return Ordering{documentsOrdering: documentsOrdering}
}

func (ordering Ordering) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return documentsWithSpamLast(ordering.documentsOrdering.OrderedDocumentsOf(answers))
}

func documentsWithSpamLast(
	orderedDocuments []queryanswers.FoundDocument,
) []queryanswers.FoundDocument {
	otherDocuments := make([]queryanswers.FoundDocument, 0, len(orderedDocuments))
	var spamDocuments []queryanswers.FoundDocument
	for _, orderedDocument := range orderedDocuments {
		if orderedDocument.SpamVerdict == spamassessment.Spam {
			spamDocuments = append(spamDocuments, orderedDocument)
			continue
		}
		otherDocuments = append(otherDocuments, orderedDocument)
	}

	return append(otherDocuments, spamDocuments...)
}
