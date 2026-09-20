// Package queryanswers holds what a spread answered for one whole query: the
// words of the query, one found document for each document it found, and how
// many documents the peers hold per query word. The text of a document, once a
// node read its page, replaces what the peers counted for it, their snippet, and
// the links they counted.
// The title of the page replaces theirs when the page has one, and the address
// the page moved to replaces theirs when it moved. A document can leave the
// answers, for example once its page is gone.
package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredQuery struct {
	QueryWords                []yacymodel.Hash
	FoundDocuments            []FoundDocument
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int
}

func (a AnsweredQuery) SaturatedWith(
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) AnsweredQuery {
	if len(pageContentsPerDocument) == 0 {
		return a
	}

	foundDocuments := make([]FoundDocument, 0, len(a.FoundDocuments))
	for _, foundDocument := range a.FoundDocuments {
		pageContents, read := pageContentsPerDocument[foundDocument.Hash]
		if read {
			foundDocument = foundDocument.saturatedWith(pageContents)
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
