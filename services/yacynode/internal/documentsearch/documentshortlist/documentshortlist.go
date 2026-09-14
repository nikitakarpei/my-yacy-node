// Package documentshortlist keeps the most relevant documents one search found,
// the best first, and drops what no longer fits. It answers the search pass with
// those documents in order, and with the lowest relevance it still holds, which
// the read of the index compares against the postings it has not read yet.
package documentshortlist

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchrelevance"
)

type Shortlist struct {
	documents  []RankedDocument
	maxResults int
}

func New(maxResults int) *Shortlist {
	return &Shortlist{maxResults: maxResults}
}

func (s *Shortlist) Place(document RankedDocument) {
	documentAt, _ := slices.BinarySearchFunc(s.documents, document, compareInRelevanceOrder)
	if s.maxResults > 0 && documentAt >= s.maxResults {
		return
	}
	s.documents = slices.Insert(s.documents, documentAt, document)
	if s.maxResults > 0 && len(s.documents) > s.maxResults {
		s.documents = s.documents[:s.maxResults]
	}
}

func (s *Shortlist) IsFull() bool {
	if s.maxResults <= 0 {
		return false
	}

	return len(s.documents) >= s.maxResults
}

func (s *Shortlist) LowestRelevance() searchrelevance.Relevance {
	if len(s.documents) == 0 {
		return 0
	}

	return s.documents[len(s.documents)-1].Relevance
}

func (s *Shortlist) InRelevanceOrder() []RankedDocument {
	return slices.Clone(s.documents)
}
