// Package queryfindings holds what a spread found for one whole query: its
// words, its compound words, the documents it found with what the peers sent
// for each of them, and how many documents the peers hold per query word. The facts of a document
// count its query words, its query phrases, its words and its links. A page the
// service read replaces the facts, the snippet, the title and the address, and
// gives the spam verdict.
package queryfindings

import (
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Findings struct {
	QueryWords                []yacymodel.Hash
	CompoundWords             []searchquery.CompoundWord
	FoundDocuments            []FoundDocument
	DocumentsHeldPerQueryWord map[yacymodel.Hash]int
}

func (f Findings) WithReadPages(
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
) Findings {
	if len(pageContentsPerDocument) == 0 {
		return f
	}

	foundDocuments := make([]FoundDocument, 0, len(f.FoundDocuments))
	for _, foundDocument := range f.FoundDocuments {
		pageContents, read := pageContentsPerDocument[foundDocument.Hash]
		if read {
			foundDocument = foundDocument.withItsReadPage(pageContents)
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	f.FoundDocuments = foundDocuments

	return f
}

func (f Findings) WithSpamVerdicts(
	spamVerdictPerDocument map[yacymodel.URLHash]spamassessment.Verdict,
) Findings {
	foundDocuments := make([]FoundDocument, 0, len(f.FoundDocuments))
	for _, foundDocument := range f.FoundDocuments {
		foundDocument.SpamVerdict = spamVerdictPerDocument[foundDocument.Hash]
		foundDocuments = append(foundDocuments, foundDocument)
	}
	f.FoundDocuments = foundDocuments

	return f
}

func (f Findings) WithoutDocuments(documents map[yacymodel.URLHash]struct{}) Findings {
	if len(documents) == 0 {
		return f
	}

	foundDocuments := make([]FoundDocument, 0, len(f.FoundDocuments))
	for _, foundDocument := range f.FoundDocuments {
		if _, left := documents[foundDocument.Hash]; left {
			continue
		}
		foundDocuments = append(foundDocuments, foundDocument)
	}
	f.FoundDocuments = foundDocuments

	return f
}
