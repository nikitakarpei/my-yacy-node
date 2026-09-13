// Package searchrelevance weighs one search: how rare each of its terms is in
// this node, how relevant a document that holds those terms is, and the largest
// relevance a posting not read yet can still reach. It answers the search pass
// with the value that orders the documents it found.
package searchrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type Relevance float64

func RelevanceOf(postings []yacymodel.RWIPosting, terms SearchTerms) Relevance {
	relevance := Relevance(0)
	for _, posting := range postings {
		relevance += Relevance(
			terms.rarityOf(posting.WordHash) *
				float64(rwipostingimpactorder.ImpactOf(posting)),
		)
	}

	return relevance
}
