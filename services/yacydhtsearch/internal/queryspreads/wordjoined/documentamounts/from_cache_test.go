package documentamounts_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentamounts"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

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

type uncachedAmounts struct {
	amounts       map[yacymodel.Hash]int
	amountOfCalls int
}

func (uncached *uncachedAmounts) AmountsInAPartitionFor(
	context.Context,
	searchquery.Query,
	leadingword.WordAsks,
) map[yacymodel.Hash]int {
	uncached.amountOfCalls++

	return uncached.amounts
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

func TestEveryWordCachedGivesTheCachedAmounts(t *testing.T) {
	t.Parallel()

	uncached := &uncachedAmounts{}
	reads := &readsFromCache{}
	fromCache := documentamounts.NewFromCache(
		cachedAmountsInAPartition(amountOfEachWord(map[string]int{firstWord: 9, secondWord: 4})),
		uncached,
		reads,
	)

	amounts := fromCache.AmountsInAPartitionFor(t.Context(), query, nil)

	if want := amountOfEachWord(map[string]int{firstWord: 9, secondWord: 4}); !maps.Equal(
		amounts, want,
	) || uncached.amountOfCalls != 0 {
		t.Fatalf(
			"the cache gave %v after %d uncached calls, want %v and no uncached call",
			amounts, uncached.amountOfCalls, want,
		)
	}
	want := []documentamounts.PerformedFromCache{{
		AmountOfQueryWords: 2, AmountOfQueryWordsCached: 2, AllQueryWordsCached: true,
	}}
	if !slices.Equal(reads.performed, want) {
		t.Fatalf("the cache reported %v, want %v", reads.performed, want)
	}
}

func TestAWordMissingFromTheCacheGivesTheUncachedAmounts(t *testing.T) {
	t.Parallel()

	uncached := &uncachedAmounts{amounts: amountOfEachWord(map[string]int{secondWord: 1})}
	reads := &readsFromCache{}
	fromCache := documentamounts.NewFromCache(
		cachedAmountsInAPartition(amountOfEachWord(map[string]int{firstWord: 9})),
		uncached,
		reads,
	)

	amounts := fromCache.AmountsInAPartitionFor(t.Context(), query, nil)

	if !maps.Equal(amounts, uncached.amounts) || uncached.amountOfCalls != 1 {
		t.Fatalf("the cache gave %v, want the uncached amounts %v", amounts, uncached.amounts)
	}
	want := []documentamounts.PerformedFromCache{{
		AmountOfQueryWords: 2, AmountOfQueryWordsCached: 1,
	}}
	if !slices.Equal(reads.performed, want) {
		t.Fatalf("the cache reported %v, want %v", reads.performed, want)
	}
}
