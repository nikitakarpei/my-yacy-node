package documentamounts_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	firstWord              = "berlin"
	secondWord             = "weather"
	twoPartitionsOfTheRing = 2
)

var query = queryreading.QueryFrom(firstWord+" "+secondWord, yacymodel.Language{})

type cachedAmountsInAPartition map[yacymodel.Hash]int

func (cached cachedAmountsInAPartition) DocumentAmountsOf(
	_ context.Context,
	words []yacymodel.Hash,
) map[yacymodel.Hash]int {
	amounts := map[yacymodel.Hash]int{}
	for _, word := range words {
		if amount, held := cached[word]; held {
			amounts[word] = amount
		}
	}

	return amounts
}

type readsFromCache struct {
	performed []documentamounts.PerformedFromCache
}

func (reads *readsFromCache) AmountsReadFromCache(
	_ context.Context,
	performed documentamounts.PerformedFromCache,
) {
	reads.performed = append(reads.performed, performed)
}

func amountOfEachWord(amounts map[string]int) map[yacymodel.Hash]int {
	amountsPerWord := make(map[yacymodel.Hash]int, len(amounts))
	for spelledWord, amount := range amounts {
		amountsPerWord[yacymodel.WordHash(spelledWord)] = amount
	}

	return amountsPerWord
}

func TestEveryWordCachedGivesTheCachedAmounts(t *testing.T) {
	t.Parallel()

	reads := &readsFromCache{}
	fromCache := documentamounts.NewFromCache(
		cachedAmountsInAPartition(amountOfEachWord(map[string]int{firstWord: 9, secondWord: 4})),
		reads,
	)

	amounts := fromCache.AmountsInAPartitionFor(t.Context(), query)

	if want := amountOfEachWord(map[string]int{firstWord: 9, secondWord: 4}); !maps.Equal(
		amounts, want,
	) {
		t.Fatalf("the cache gave %v, want %v", amounts, want)
	}
	want := []documentamounts.PerformedFromCache{{
		AmountOfQueryWords: 2, AmountOfQueryWordsCached: 2, AllQueryWordsCached: true,
	}}
	if !slices.Equal(reads.performed, want) {
		t.Fatalf("the cache reported %v, want %v", reads.performed, want)
	}
}

func TestAWordMissingFromTheCacheGivesNoAmounts(t *testing.T) {
	t.Parallel()

	reads := &readsFromCache{}
	fromCache := documentamounts.NewFromCache(
		cachedAmountsInAPartition(amountOfEachWord(map[string]int{firstWord: 9})),
		reads,
	)

	amounts := fromCache.AmountsInAPartitionFor(t.Context(), query)

	if len(amounts) != 0 {
		t.Fatalf("the cache gave %v, want no amounts", amounts)
	}
	want := []documentamounts.PerformedFromCache{{
		AmountOfQueryWords: 2, AmountOfQueryWordsCached: 1,
	}}
	if !slices.Equal(reads.performed, want) {
		t.Fatalf("the cache reported %v, want %v", reads.performed, want)
	}
}
