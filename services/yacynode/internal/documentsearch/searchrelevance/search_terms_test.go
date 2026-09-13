package searchrelevance_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
)

func TestTheRarestTermIsTheTermWithTheFewestPostings(t *testing.T) {
	first, second, third := searchtest.HashFor("w1"),
		searchtest.HashFor("w2"),
		searchtest.HashFor("w3")
	terms := searchTermsOf(
		[]yacymodel.Hash{first, second, third},
		map[yacymodel.Hash]int{first: 100, second: 3, third: 40},
	)

	if terms.RarestTerm() != second {
		t.Errorf("rarest term = %v, want %v", terms.RarestTerm(), second)
	}
	if !slices.Equal(terms.OtherTerms(), []yacymodel.Hash{first, third}) {
		t.Errorf(
			"other terms = %v, want the remaining terms in the order the search names them",
			terms.OtherTerms(),
		)
	}
}

func TestTheFirstTermIsTheRarestWhenTwoTermsHoldAsManyPostings(t *testing.T) {
	first, second := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	terms := searchTermsOf(
		[]yacymodel.Hash{first, second},
		map[yacymodel.Hash]int{first: 7, second: 7},
	)

	if terms.RarestTerm() != first {
		t.Errorf("rarest term = %v, want the first term %v", terms.RarestTerm(), first)
	}
	if !slices.Equal(terms.OtherTerms(), []yacymodel.Hash{second}) {
		t.Errorf("other terms = %v, want %v", terms.OtherTerms(), second)
	}
}
