package documentmatch

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type documentRanker struct {
	terms                              searchTerms
	largestRelevanceFromOtherTerms     float64
	maxTermSpread                      int
	maxResults                         int
	rankedDocuments                    []rankedDocument
	amountOfDocumentsMatchingEveryTerm int
}

func (r *documentRanker) rarestTerm() yacymodel.Hash {
	return r.terms.rarestTerm
}

func (r *documentRanker) hasReachedRelevanceBoundAt(
	impactOfNextUnreadPosting rwipostingimpactorder.Impact,
) bool {
	if r.maxResults <= 0 || len(r.rankedDocuments) < r.maxResults {
		return false
	}

	return r.rankedDocuments[len(r.rankedDocuments)-1].relevance >=
		r.largestRelevanceReachableFrom(impactOfNextUnreadPosting)
}

func (r *documentRanker) largestRelevanceReachableFrom(
	impactOfNextUnreadPosting rwipostingimpactorder.Impact,
) float64 {
	return r.terms.rarityOf(r.rarestTerm())*float64(impactOfNextUnreadPosting) +
		r.largestRelevanceFromOtherTerms
}

func (r *documentRanker) rankPostingsOfDocument(postings []yacymodel.RWIPosting) {
	document := rankedDocument{
		joinedPosting: joinedPostingOf(postings),
		relevance:     relevanceOf(postings, r.terms),
		termSpread:    termSpreadOf(postings),
	}
	if !r.isWithinTermSpread(document) {
		return
	}
	r.amountOfDocumentsMatchingEveryTerm++
	r.placeRankedDocument(document)
}

func (r *documentRanker) isWithinTermSpread(document rankedDocument) bool {
	if r.maxTermSpread <= 0 {
		return true
	}

	return document.termSpread <= r.maxTermSpread
}

func (r *documentRanker) placeRankedDocument(document rankedDocument) {
	documentAt, _ := slices.BinarySearchFunc(r.rankedDocuments, document, r.compare)
	if r.maxResults > 0 && documentAt >= r.maxResults {
		return
	}
	r.rankedDocuments = slices.Insert(r.rankedDocuments, documentAt, document)
	if r.maxResults > 0 && len(r.rankedDocuments) > r.maxResults {
		r.rankedDocuments = r.rankedDocuments[:r.maxResults]
	}
}

func (r *documentRanker) compare(a, b rankedDocument) int {
	return cmp.Or(
		cmp.Compare(b.relevance, a.relevance),
		cmp.Compare(a.termSpread, b.termSpread),
		yacymodel.CompareInAlphabetOrder(
			a.joinedPosting.URLHash.String(),
			b.joinedPosting.URLHash.String(),
		),
	)
}

func (r *documentRanker) joinedPostingsInRelevanceOrder() []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(r.rankedDocuments))
	for _, document := range r.rankedDocuments {
		postings = append(postings, document.joinedPosting)
	}

	return postings
}
