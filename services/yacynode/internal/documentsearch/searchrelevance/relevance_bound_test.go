package searchrelevance_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

func TestTheBoundAddsTheLargestPostingOfEveryOtherTermToTheRarestTerm(t *testing.T) {
	rareTerm, commonTerm := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	terms := searchTermsOf(
		[]yacymodel.Hash{rareTerm, commonTerm},
		map[yacymodel.Hash]int{rareTerm: 2, commonTerm: 60},
	)
	nextUnreadPosting := postingOf(rareTerm, "u1", 5)
	largestPostingOfCommonTerm := titlePostingOf(commonTerm, "u2")

	bound := searchrelevance.RelevanceBoundOf(
		terms,
		map[yacymodel.Hash]rwipostingimpactorder.Impact{
			commonTerm: rwipostingimpactorder.ImpactOf(largestPostingOfCommonTerm),
		},
	)

	largestRelevance := bound.LargestRelevanceReachableFrom(
		rwipostingimpactorder.ImpactOf(nextUnreadPosting),
	)
	reachableRelevance := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{nextUnreadPosting, largestPostingOfCommonTerm},
		terms,
	)
	if largestRelevance != reachableRelevance {
		t.Errorf(
			"largest relevance = %v, want %v of a document holding both postings",
			largestRelevance,
			reachableRelevance,
		)
	}
}

func TestATermTheNodeHoldsNoPostingOfRaisesTheBoundByNothing(t *testing.T) {
	rareTerm, absentTerm := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	terms := searchTermsOf(
		[]yacymodel.Hash{rareTerm, absentTerm},
		map[yacymodel.Hash]int{rareTerm: 2, absentTerm: 60},
	)
	nextUnreadPosting := postingOf(rareTerm, "u1", 5)

	bound := searchrelevance.RelevanceBoundOf(
		terms,
		map[yacymodel.Hash]rwipostingimpactorder.Impact{},
	)

	largestRelevance := bound.LargestRelevanceReachableFrom(
		rwipostingimpactorder.ImpactOf(nextUnreadPosting),
	)
	reachableRelevance := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{nextUnreadPosting},
		terms,
	)
	if largestRelevance != reachableRelevance {
		t.Errorf(
			"largest relevance = %v, want %v of the rarest term alone",
			largestRelevance,
			reachableRelevance,
		)
	}
}
