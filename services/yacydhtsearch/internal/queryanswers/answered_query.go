// Package queryanswers holds what a spread answered for one whole query: its
// words, its compound words, which are two adjacent query words spelled as one,
// the documents it found with what the peers sent for each of them, and how
// many documents the peers hold per query word. The facts of a document
// count its query words, its query phrases, its words and its links. A page the
// service read replaces the facts, the snippet, the title and the address.
package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredQuery struct {
	QueryWords                []yacymodel.Hash
	CompoundQueryWords        []CompoundQueryWord
	FoundDocuments            []FoundDocument
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int
}

func (a AnsweredQuery) WithReadPages(
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) AnsweredQuery {
	if len(pageContentsPerDocument) == 0 {
		return a
	}

	foundDocuments := make([]FoundDocument, 0, len(a.FoundDocuments))
	for _, foundDocument := range a.FoundDocuments {
		pageContents, read := pageContentsPerDocument[foundDocument.Hash]
		if read {
			foundDocument = foundDocument.withItsReadPage(pageContents)
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	a.FoundDocuments = foundDocuments

	return a
}

func (a AnsweredQuery) WithoutDocuments(documents map[yacymodel.URLHash]struct{}) AnsweredQuery {
	if len(documents) == 0 {
		return a
	}

	foundDocuments := make([]FoundDocument, 0, len(a.FoundDocuments))
	for _, foundDocument := range a.FoundDocuments {
		if _, left := documents[foundDocument.Hash]; left {
			continue
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	a.FoundDocuments = foundDocuments

	return a
}
