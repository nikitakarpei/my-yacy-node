// Package queryanswers holds what a spread answered for one whole query: the
// words of the query, one found document for each document it found, and how
// many documents the peers hold per query word. The text of a document, once a
// node read its page, replaces what the peers counted for it and their snippet,
// and the title of the page replaces theirs when the page has one. A document
// can leave the answers, for example once its page is gone.
package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredQuery struct {
	QueryWords                []yacymodel.Hash
	FoundDocuments            []FoundDocument
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int
}

func (a AnsweredQuery) SaturatedWith(
	textPerDocument map[yacymodel.URLHash]documenttext.DocumentText,
) AnsweredQuery {
	if len(textPerDocument) == 0 {
		return a
	}

	foundDocuments := make([]FoundDocument, 0, len(a.FoundDocuments))
	for _, foundDocument := range a.FoundDocuments {
		text, read := textPerDocument[foundDocument.Hash]
		if read {
			foundDocument = foundDocument.saturatedWith(text)
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
