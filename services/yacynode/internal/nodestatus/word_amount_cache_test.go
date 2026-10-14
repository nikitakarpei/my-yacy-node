package nodestatus_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
)

const wordAmountLifetime = 5 * time.Minute

type changingEstimate struct {
	words int
	err   error
}

func (e *changingEstimate) EstimatedAmountOfWords(*vault.Txn) (int, error) {
	return e.words, e.err
}

type fakeClock struct {
	instant time.Time
}

func (c *fakeClock) now() time.Time { return c.instant }

func (c *fakeClock) advance(elapsed time.Duration) { c.instant = c.instant.Add(elapsed) }

func cachedWordAmountOf(t *testing.T, cache nodestatus.WordAmountEstimate) int {
	t.Helper()

	amountOfWords, err := cache.EstimatedAmountOfWords(nil)
	if err != nil {
		t.Fatalf("EstimatedAmountOfWords: %v", err)
	}

	return amountOfWords
}

func TestAWordAmountCacheServesItsEstimateUntilTheLifetimeEnds(t *testing.T) {
	estimate := &changingEstimate{words: 7}
	clock := &fakeClock{instant: time.Date(2026, time.October, 14, 10, 0, 0, 0, time.UTC)}
	cache := nodestatus.NewWordAmountCache(estimate, clock.now, wordAmountLifetime)

	cachedWordAmountOf(t, cache)
	estimate.words = 9
	clock.advance(wordAmountLifetime - time.Second)
	if amountOfWords := cachedWordAmountOf(t, cache); amountOfWords != 7 {
		t.Fatalf("amount of words within the lifetime = %d, want the cached 7", amountOfWords)
	}
	clock.advance(time.Second)
	if amountOfWords := cachedWordAmountOf(t, cache); amountOfWords != 9 {
		t.Fatalf("amount of words after the lifetime = %d, want a new estimate of 9", amountOfWords)
	}
}

func TestAWordAmountCacheKeepsNoFailedEstimate(t *testing.T) {
	estimate := &changingEstimate{err: errors.New("boom")}
	clock := &fakeClock{instant: time.Date(2026, time.October, 14, 10, 0, 0, 0, time.UTC)}
	cache := nodestatus.NewWordAmountCache(estimate, clock.now, wordAmountLifetime)

	if _, err := cache.EstimatedAmountOfWords(nil); err == nil {
		t.Fatal("EstimatedAmountOfWords hid the failed estimate, want its error")
	}
	estimate.words, estimate.err = 4, nil
	if amountOfWords := cachedWordAmountOf(t, cache); amountOfWords != 4 {
		t.Fatalf(
			"amount of words after a failed estimate = %d, want a new estimate of 4",
			amountOfWords,
		)
	}
}
