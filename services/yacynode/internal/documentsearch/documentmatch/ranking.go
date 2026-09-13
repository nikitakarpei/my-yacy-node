package documentmatch

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type ranking struct {
	terms                              termsByRarity
	relevanceBoundOfOtherTerms         float64
	maxTermSpread                      int
	maxResults                         int
	rankedDocuments                    []rankedDocument
	amountOfDocumentsMatchingEveryTerm int
}

func rankingFor(
	criteria searchcriteria.Criteria,
	terms termsByRarity,
	relevanceBoundOfOtherTerms float64,
) ranking {
	return ranking{
		terms:                      terms,
		relevanceBoundOfOtherTerms: relevanceBoundOfOtherTerms,
		maxTermSpread:              criteria.MaxTermSpread,
		maxResults:                 criteria.MaxResults,
	}
}

func (r *ranking) rarestTerm() yacymodel.Hash {
	return r.terms.rarestTerm
}

func (r *ranking) hasReachedRelevanceBoundAt(
	impactOfNextUnreadPosting rwipostingimpactorder.Impact,
) bool {
	if r.maxResults <= 0 || len(r.rankedDocuments) < r.maxResults {
		return false
	}

	return r.rankedDocuments[len(r.rankedDocuments)-1].relevance >=
		r.largestRelevanceReachableFrom(impactOfNextUnreadPosting)
}

func (r *ranking) largestRelevanceReachableFrom(
	impactOfNextUnreadPosting rwipostingimpactorder.Impact,
) float64 {
	return r.terms.rarityOf(r.rarestTerm())*float64(impactOfNextUnreadPosting) +
		r.relevanceBoundOfOtherTerms
}

func (r *ranking) rankPostingsOfDocument(postings []yacymodel.RWIPosting) {
	document := rankedDocument{
		posting:    mergedPostingOf(postings),
		relevance:  relevanceOf(postings, r.terms),
		termSpread: termSpreadOf(postings),
	}
	if !r.isWithinTermSpread(document) {
		return
	}
	r.amountOfDocumentsMatchingEveryTerm++
	r.placeRankedDocument(document)
}

func (r *ranking) isWithinTermSpread(document rankedDocument) bool {
	if r.maxTermSpread <= 0 {
		return true
	}

	return document.termSpread <= r.maxTermSpread
}

func (r *ranking) placeRankedDocument(document rankedDocument) {
	documentAt, _ := slices.BinarySearchFunc(r.rankedDocuments, document, r.compare)
	if r.maxResults > 0 && documentAt >= r.maxResults {
		return
	}
	r.rankedDocuments = slices.Insert(r.rankedDocuments, documentAt, document)
	if r.maxResults > 0 && len(r.rankedDocuments) > r.maxResults {
		r.rankedDocuments = r.rankedDocuments[:r.maxResults]
	}
}

func (r *ranking) compare(a, b rankedDocument) int {
	return cmp.Or(
		cmp.Compare(b.relevance, a.relevance),
		cmp.Compare(a.termSpread, b.termSpread),
		yacymodel.CompareInAlphabetOrder(
			a.posting.URLHash.String(),
			b.posting.URLHash.String(),
		),
	)
}

func (r *ranking) postingsInRelevanceOrder() []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(r.rankedDocuments))
	for _, document := range r.rankedDocuments {
		postings = append(postings, document.posting)
	}

	return postings
}
