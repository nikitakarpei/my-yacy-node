package vaultkey_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestEveryKeyIsUnboundedOnBothSides(t *testing.T) {
	firstIncluded, firstExcluded := vaultkey.EveryKey().Bounds()

	if firstIncluded != nil {
		t.Fatalf("EveryKey lower bound = %x, want nil", firstIncluded)
	}
	if firstExcluded != nil {
		t.Fatalf("EveryKey upper bound = %x, want nil", firstExcluded)
	}
}

func TestKeysFromHoldsItsBoundAndIsUnboundedAbove(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Integer)
	keys := vaultkey.KeysFrom(layout.Key(10))

	firstIncluded, firstExcluded := keys.Bounds()
	if !bytes.Equal(firstIncluded, layout.Key(10).Bytes()) {
		t.Fatalf("KeysFrom lower bound = %x, want %x", firstIncluded, layout.Key(10).Bytes())
	}
	if firstExcluded != nil {
		t.Fatalf("KeysFrom upper bound = %x, want nil", firstExcluded)
	}
	if !holdsKey(keys, layout.Key(10).Bytes()) {
		t.Fatal("KeysFrom excludes its bound")
	}
	if holdsKey(keys, layout.Key(9).Bytes()) {
		t.Fatal("KeysFrom holds a smaller key")
	}
	if !holdsKey(keys, layout.Key(math.MaxInt64).Bytes()) {
		t.Fatal("KeysFrom is bounded above")
	}
}

func TestRangesOfTheFirstPositionReadInDomainOrderInBothDirections(t *testing.T) {
	const bound = int64(2)

	for _, directed := range []struct {
		direction string
		layout    vaultkey.PairKey[int64, string]
	}{
		{"Ascending", vaultkey.Pair(vaultkey.Integer, vaultkey.Text)},
		{"Descending", vaultkey.Pair(vaultkey.IntegerDescending, vaultkey.Text)},
	} {
		t.Run(directed.direction, func(t *testing.T) {
			for _, first := range []int64{math.MinInt64, 1, 2, 3, math.MaxInt64} {
				for _, second := range []string{"", "\xff\xff"} {
					key := directed.layout.Key(first, second).Bytes()

					assertRangeAnswers(
						t, directed.layout.KeysWithFirst(bound), key, first == bound)
					assertRangeAnswers(
						t, directed.layout.KeysThroughFirst(bound), key, first <= bound)
					assertRangeAnswers(
						t, directed.layout.KeysBeforeFirst(bound), key, first < bound)
				}
			}
		})
	}
}

func TestKeysBeforeTheSmallestFirstValueHoldNoKey(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Integer, vaultkey.Text)
	keys := layout.KeysBeforeFirst(math.MinInt64)

	for _, first := range []int64{math.MinInt64, 0, math.MaxInt64} {
		assertRangeAnswers(t, keys, layout.Key(first, "").Bytes(), false)
	}
}

func assertRangeAnswers(t *testing.T, keys vaultkey.KeyRange, key []byte, want bool) {
	t.Helper()

	if got := holdsKey(keys, key); got != want {
		t.Fatalf("range holds %x = %v, want %v", key, got, want)
	}
}

func holdsKey(keys vaultkey.KeyRange, key []byte) bool {
	firstIncluded, firstExcluded := keys.Bounds()
	if bytes.Compare(key, firstIncluded) < 0 {
		return false
	}

	return firstExcluded == nil || bytes.Compare(key, firstExcluded) < 0
}
