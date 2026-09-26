package documentrelevance_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
)

func answersOfCompoundWords(
	queryWords []string,
	foundDocuments []foundDocument,
	documentsHeldPerWord map[string]int,
) queryanswers.AnsweredQuery {
	answers := answersHoldingDocumentsPerQueryWord(queryWords, foundDocuments, documentsHeldPerWord)
	answers.CompoundWords = queryreading.QueryFrom(strings.Join(queryWords, " "), "").CompoundWords

	return answers
}

func TestTitleThatSpellsTwoAdjacentQueryWordsAsOneHoldsBoth(t *testing.T) {
	t.Parallel()

	answers := answersOfCompoundWords(
		[]string{"arch", "wiki"},
		[]foundDocument{
			foundDocumentAt(t, "https://beside.example/").
				matchingWords("arch", "wiki").
				withTitle("Arch Linux Forums"),
			foundDocumentAt(t, "https://compound.example/").
				matchingWords("arch", "wiki").
				withTitle("Installation guide - ArchWiki"),
		},
		map[string]int{"arch": 100, "wiki": 100},
	)

	want := []string{"https://compound.example/", "https://beside.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestSiteNameThatSpellsTwoAdjacentQueryWordsAsOneHoldsBoth(t *testing.T) {
	t.Parallel()

	answers := answersOfCompoundWords(
		[]string{"self", "hosted"},
		[]foundDocument{
			foundDocumentAt(t, "https://hosted.example/").matchingWords("self", "hosted"),
			foundDocumentAt(t, "https://selfhosted.example/").matchingWords("self", "hosted"),
		},
		map[string]int{"self": 100, "hosted": 100},
	)

	want := []string{"https://selfhosted.example/", "https://hosted.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTitleThatSpellsQueryWordsInTheOtherOrderAsOneHoldsNeither(t *testing.T) {
	t.Parallel()

	answers := answersOfCompoundWords(
		[]string{"arch", "wiki"},
		[]foundDocument{
			foundDocumentAt(t, "https://titled.example/").
				matchingWords("arch", "wiki").
				withTitle("Arch Linux"),
			foundDocumentAt(t, "https://reversed.example/").
				matchingWords("arch", "wiki").
				withTitle("WikiArch"),
		},
		map[string]int{"arch": 100, "wiki": 100},
	)

	want := []string{"https://titled.example/", "https://reversed.example/"}
	if got := addressesInFallingOrderOfRelevance(answers); !slices.Equal(got, want) {
		t.Fatalf("the relevance order reads %v, want %v", got, want)
	}
}

func TestTitleThatHoldsTheWholeQueryOutweighsTwoTitlesOfHalfTheQueryEach(t *testing.T) {
	t.Parallel()

	answers := answersOfCompoundWords(
		[]string{"linux", "kernel"},
		[]foundDocument{
			foundDocumentAt(t, "https://half.example/").
				matchingWords("linux", "kernel").
				withTitle("The kernel").
				withHitsOf("linux", 20).
				withHitsOf("kernel", 20),
			foundDocumentAt(t, "https://whole.example/").
				matchingWords("linux", "kernel").
				withTitle("Linux kernel"),
		},
		map[string]int{"linux": 100, "kernel": 100},
	)

	whole := relevanceOfDocumentAt(t, answers, "https://whole.example/")
	half := relevanceOfDocumentAt(t, answers, "https://half.example/")
	if whole < 2*half {
		t.Fatalf("the whole title scores %.2f against %.2f, want at least twice", whole, half)
	}
}
