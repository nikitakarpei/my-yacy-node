package leadingword_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord  = "berlin"
	secondWord = "weather"
)

var query = queryreading.QueryFrom(firstWord+" "+secondWord, "")

type amountsInAPartition map[string]int

func (amounts amountsInAPartition) AmountsInAPartitionFor(
	context.Context,
	searchquery.Query,
	leadingword.WordAsks,
) map[yacymodel.Hash]int {
	amountOfEachWord := make(map[yacymodel.Hash]int, len(amounts))
	for spelledWord, amount := range amounts {
		amountOfEachWord[yacymodel.WordHash(spelledWord)] = amount
	}

	return amountOfEachWord
}

func leadFrom(t *testing.T, amounts amountsInAPartition) leadingword.Lead {
	t.Helper()

	return leadingword.New(amounts).FindFor(t.Context(), query, nil)
}

func TestTheWordWithTheFewestDocumentsLeadsWithItsAmount(t *testing.T) {
	t.Parallel()

	lead := leadFrom(t, amountsInAPartition{firstWord: 3, secondWord: 1})

	want := leadingword.Lead{
		Word:                          yacymodel.Some(yacymodel.WordHash(secondWord)),
		AmountOfDocumentsInAPartition: yacymodel.Some(1),
	}
	if lead != want {
		t.Fatalf("the lead is %+v, want %+v", lead, want)
	}
}

func TestOfWordsWithAsManyDocumentsTheFirstQueryWordLeads(t *testing.T) {
	t.Parallel()

	lead := leadFrom(t, amountsInAPartition{firstWord: 2, secondWord: 2})

	if word, _ := lead.Word.Get(); word != yacymodel.WordHash(firstWord) {
		t.Fatalf("the lead is %+v, want the first query word", lead)
	}
}

func TestOnlyAWordWithAnAmountCanLead(t *testing.T) {
	t.Parallel()

	lead := leadFrom(t, amountsInAPartition{secondWord: 7})

	if word, _ := lead.Word.Get(); word != yacymodel.WordHash(secondWord) {
		t.Fatalf("the lead is %+v, want the only word with an amount", lead)
	}
}

func TestWithoutAnAmountNoWordLeads(t *testing.T) {
	t.Parallel()

	lead := leadFrom(t, amountsInAPartition{})

	if lead.Word.Present() || lead.AmountOfDocumentsInAPartition.Present() {
		t.Fatalf("the lead is %+v, want no word and no amount", lead)
	}
}
