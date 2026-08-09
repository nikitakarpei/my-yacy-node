package vaultkey_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestEveryKeyIsUnboundedOnBothSides(t *testing.T) {
	keys := vaultkey.EveryKey()

	if keys.FirstIncludedKey() != nil {
		t.Fatalf("EveryKey lower bound = %x, want nil", keys.FirstIncludedKey())
	}
	for _, key := range [][]byte{nil, {}, []byte("a"), {0xFF, 0xFF}} {
		if !keys.Contains(key) {
			t.Fatalf("EveryKey excludes %x", key)
		}
	}
}

func TestKeysFromHoldsItsBoundAndIsUnboundedAbove(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Integer)
	keys := vaultkey.KeysFrom(layout.Key(10))

	if !keys.Contains(layout.Key(10).Bytes()) {
		t.Fatal("KeysFrom excludes its bound")
	}
	if keys.Contains(layout.Key(9).Bytes()) {
		t.Fatal("KeysFrom holds a smaller key")
	}
	if !keys.Contains(layout.Key(math.MaxInt64).Bytes()) {
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

func assertRangeAnswers(t *testing.T, keys vaultkey.KeyRange, key []byte, want bool) {
	t.Helper()

	if got := keys.Contains(key); got != want {
		t.Fatalf("Contains(%x) = %v, want %v", key, got, want)
	}
}
