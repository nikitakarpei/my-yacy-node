// Package queryanswers holds what a spread answered for one whole query: the
// words of the query, one found document for each document it found, the
// posting each holder sent for it, the facts counted for each of them, and how
// many documents the peers hold per query word. The facts of a document come
// from the first posting of each query word, until a node reads its page. The facts of a document are how often its text holds each query word
// and each query phrase, how many words it holds, and how many links.
// The facts a node counted from the page it read replace the facts the peers
// counted, a query word the text holds none of included, and no peer counts a
// query phrase. The snippet of the page replaces theirs, the title of the page
// replaces theirs when the page has one, and the address the page moved to
// replaces theirs when it moved. A document can leave the answers, for example
// once its page is gone.
package queryanswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredQuery struct {
	QueryWords                 []yacymodel.Hash
	FoundDocuments             []FoundDocument
	PostingReplicasPerDocument PostingReplicasPerDocument
	FactsPerDocument           FactsPerDocument
	DocumentsHeldPerQueryWord  map[yacymodel.Hash]int
}

func (a AnsweredQuery) WithTheContentsOfTheReadPages(
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) AnsweredQuery {
	if len(pageContentsPerDocument) == 0 {
		return a
	}

	foundDocuments := make([]FoundDocument, 0, len(a.FoundDocuments))
	for _, foundDocument := range a.FoundDocuments {
		pageContents, read := pageContentsPerDocument[foundDocument.Hash]
		if read {
			foundDocument = foundDocument.withTheContentsOfItsReadPage(pageContents)
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	a.FoundDocuments = foundDocuments
	a.FactsPerDocument = a.FactsPerDocument.withTheFactsOfTheReadPages(pageContentsPerDocument)

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
	a.PostingReplicasPerDocument = a.PostingReplicasPerDocument.withoutDocuments(documents)
	a.FactsPerDocument = a.FactsPerDocument.withoutDocuments(documents)

	return a
}
