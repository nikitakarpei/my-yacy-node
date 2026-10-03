package wordroles_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord  = "berlin"
	secondWord = "weather"
	thirdWord  = "rain"
)

func TestACompoundWordHoldingTheLeadingWordIsAListingWord(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom(firstWord+" "+secondWord+" "+thirdWord, "")
	lead := leadingword.Lead{Word: yacymodel.Some(yacymodel.WordHash(firstWord))}

	roles := wordroles.From(lead, query)

	wantedListingWords := []yacymodel.Hash{
		yacymodel.WordHash(firstWord),
		yacymodel.WordHash(firstWord + secondWord),
		yacymodel.WordHash(firstWord + secondWord + thirdWord),
	}
	wantedMatchingWords := []yacymodel.Hash{
		yacymodel.WordHash(secondWord),
		yacymodel.WordHash(thirdWord),
		yacymodel.WordHash(secondWord + thirdWord),
	}
	if !slices.Equal(roles.ListingWords, wantedListingWords) ||
		!slices.Equal(roles.MatchingWords, wantedMatchingWords) {
		t.Fatalf(
			"the roles are %+v, want %v as listing words and %v as matching words",
			roles, wantedListingWords, wantedMatchingWords,
		)
	}
}

func TestWithoutALeadEveryWordIsAListingWord(t *testing.T) {
	t.Parallel()

	roles := wordroles.From(
		leadingword.Lead{},
		queryreading.QueryFrom(firstWord+" "+secondWord, ""),
	)

	want := []yacymodel.Hash{
		yacymodel.WordHash(firstWord),
		yacymodel.WordHash(secondWord),
		yacymodel.WordHash(firstWord + secondWord),
	}
	if !slices.Equal(roles.ListingWords, want) || len(roles.MatchingWords) != 0 {
		t.Fatalf("the roles are %+v, want every word asked whole", roles)
	}
}
