package documentmatch

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiimpactorder"
)

type ranking struct {
	rarity                           wordRarity
	largestRelevanceBesideRarestWord float64
	amountOfTerms                    int
	maxTermSpread                    int
	maxResults                       int
	matches                          []documentMatch
	amountOfMatchedDocuments         int
}

func rankingFor(
	criteria searchcriteria.Criteria,
	rarity wordRarity,
	largestRelevanceBesideRarestWord float64,
) ranking {
	return ranking{
		rarity:                           rarity,
		largestRelevanceBesideRarestWord: largestRelevanceBesideRarestWord,
		amountOfTerms:                    len(criteria.Terms),
		maxTermSpread:                    criteria.MaxTermSpread,
		maxResults:                       criteria.MaxResults,
	}
}

func (r *ranking) rarestWord() yacymodel.Hash {
	return r.rarity.rarestWord()
}

func (r *ranking) noUnreadDocumentCanEnterTheAnswer(nextImpact rwiimpactorder.Impact) bool {
	if r.maxResults <= 0 || len(r.matches) < r.maxResults {
		return false
	}

	return r.matches[len(r.matches)-1].relevance >= r.largestRelevanceFrom(nextImpact)
}

func (r *ranking) largestRelevanceFrom(nextImpact rwiimpactorder.Impact) float64 {
	return r.rarity.rarityOf(r.rarestWord())*float64(nextImpact) +
		r.largestRelevanceBesideRarestWord
}

func (r *ranking) consider(postings []yacymodel.RWIPosting) {
	match := matchAcrossTerms(postings, r.rarity)
	if !r.isWithinTermSpread(match) {
		return
	}
	r.amountOfMatchedDocuments++
	r.place(match)
}

func (r *ranking) isWithinTermSpread(match documentMatch) bool {
	if r.maxTermSpread <= 0 {
		return true
	}

	return match.termSpread(r.amountOfTerms) <= r.maxTermSpread
}

func (r *ranking) place(match documentMatch) {
	position, _ := slices.BinarySearchFunc(r.matches, match, r.compare)
	if r.maxResults > 0 && position >= r.maxResults {
		return
	}
	r.matches = slices.Insert(r.matches, position, match)
	if r.maxResults > 0 && len(r.matches) > r.maxResults {
		r.matches = r.matches[:r.maxResults]
	}
}

func (r *ranking) compare(a, b documentMatch) int {
	return cmp.Or(
		cmp.Compare(b.relevance, a.relevance),
		cmp.Compare(a.termSpread(r.amountOfTerms), b.termSpread(r.amountOfTerms)),
		yacymodel.CompareInAlphabetOrder(
			a.posting.URLHash.String(),
			b.posting.URLHash.String(),
		),
	)
}

func (r *ranking) postingsInRelevanceOrder() []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(r.matches))
	for _, match := range r.matches {
		postings = append(postings, match.posting)
	}

	return postings
}
