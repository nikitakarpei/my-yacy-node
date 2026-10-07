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

func TestACompoundWordHoldingTheLeadIsACompoundWordOfTheLead(t *testing.T) {
	t.Parallel()

	query := queryreading.QueryFrom(firstWord+" "+secondWord+" "+thirdWord, yacymodel.Language{})
	lead := leadingword.Lead{Word: yacymodel.WordHash(firstWord)}

	roles := wordroles.Around(lead, query)

	wantedLeadAndItsCompoundWords := []yacymodel.Hash{
		yacymodel.WordHash(firstWord),
		yacymodel.WordHash(firstWord + secondWord),
		yacymodel.WordHash(firstWord + secondWord + thirdWord),
	}
	wantedOtherWords := []yacymodel.Hash{
		yacymodel.WordHash(secondWord),
		yacymodel.WordHash(thirdWord),
		yacymodel.WordHash(secondWord + thirdWord),
	}
	if !slices.Equal(roles.LeadAndItsCompoundWords, wantedLeadAndItsCompoundWords) ||
		!slices.Equal(roles.OtherWords, wantedOtherWords) {
		t.Fatalf(
			"the roles are %+v, want %v as the lead and its compound words and %v as the other words",
			roles,
			wantedLeadAndItsCompoundWords,
			wantedOtherWords,
		)
	}
}
