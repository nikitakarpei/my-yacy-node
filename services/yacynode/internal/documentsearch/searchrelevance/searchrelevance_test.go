package searchrelevance_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
)

func postingOf(term yacymodel.Hash, document string, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: term,
		URLHash:  searchtest.URLHashFor(document),
		Hits:     hits,
	}
}

func titlePostingOf(term yacymodel.Hash, document string) yacymodel.RWIPosting {
	posting := postingOf(term, document, 1)
	posting.Appearance.AppearsInTitle = true

	return posting
}

func searchTermsOf(
	terms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) searchrelevance.SearchTerms {
	return searchrelevance.SearchTermsOf(
		searchcriteria.Criteria{Terms: terms},
		amountOfPostingsPerTerm,
	)
}

func TestARareTermWeighsMoreThanACommonTerm(t *testing.T) {
	rareTerm, commonTerm := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	terms := searchTermsOf(
		[]yacymodel.Hash{rareTerm, commonTerm},
		map[yacymodel.Hash]int{rareTerm: 1, commonTerm: 100},
	)

	relevanceOfRareTerm := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{postingOf(rareTerm, "u1", 5)},
		terms,
	)
	relevanceOfCommonTerm := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{postingOf(commonTerm, "u1", 5)},
		terms,
	)
	if relevanceOfRareTerm <= relevanceOfCommonTerm {
		t.Errorf(
			"relevance of the rare term = %v, want more than %v of the common term",
			relevanceOfRareTerm,
			relevanceOfCommonTerm,
		)
	}
}

func TestATermInTheTitleRaisesTheRelevanceOfADocument(t *testing.T) {
	term := searchtest.HashFor("w1")
	terms := searchTermsOf([]yacymodel.Hash{term}, map[yacymodel.Hash]int{term: 10})

	relevanceOfTitle := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{titlePostingOf(term, "u1")},
		terms,
	)
	relevanceOfText := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{postingOf(term, "u2", 1)},
		terms,
	)
	if relevanceOfTitle <= relevanceOfText {
		t.Errorf(
			"relevance of the title posting = %v, want more than %v of the text posting",
			relevanceOfTitle,
			relevanceOfText,
		)
	}
}

func TestADocumentHoldingEveryTermIsMoreRelevantThanOneHoldingTheRarestAlone(t *testing.T) {
	rareTerm, commonTerm := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	terms := searchTermsOf(
		[]yacymodel.Hash{rareTerm, commonTerm},
		map[yacymodel.Hash]int{rareTerm: 1, commonTerm: 100},
	)

	relevanceOfBothTerms := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{postingOf(rareTerm, "u1", 5), postingOf(commonTerm, "u1", 5)},
		terms,
	)
	relevanceOfTheRareTerm := searchrelevance.RelevanceOf(
		[]yacymodel.RWIPosting{postingOf(rareTerm, "u1", 5)},
		terms,
	)
	if relevanceOfBothTerms <= relevanceOfTheRareTerm {
		t.Errorf(
			"relevance of both terms = %v, want more than %v of the rare term alone",
			relevanceOfBothTerms,
			relevanceOfTheRareTerm,
		)
	}
}
