package vault_test

import (
	"bytes"
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestEveryKeyIsUnboundedOnBothSides(t *testing.T) {
	firstIncluded, firstExcluded := vault.EveryKey().Bounds()

	if firstIncluded != nil {
		t.Fatalf("vault.EveryKey lower bound = %x, want nil", firstIncluded)
	}
	if firstExcluded != nil {
		t.Fatalf("vault.EveryKey upper bound = %x, want nil", firstExcluded)
	}
}

var firstValuesSpanningTheEncodings = []int64{math.MinInt64, 1, 2, 3, 1 << 62, math.MaxInt64}

func TestRangesOfTheFirstPositionReadInDomainOrderInBothDirections(t *testing.T) {
	for _, directed := range []struct {
		direction string
		parts     vault.PairKeyParts[int64, string]
	}{
		{"Ascending", vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)},
		{"Descending", vault.PairKey(vault.IntegerKeyPartDescending, vault.TextKeyPart)},
	} {
		t.Run(directed.direction, func(t *testing.T) {
			for _, bound := range firstValuesSpanningTheEncodings {
				for _, first := range firstValuesSpanningTheEncodings {
					for _, second := range []string{"", "\xff\xff"} {
						key := directed.parts.Key(first, second).Bytes()

						assertRangeAnswers(
							t, directed.parts.KeysWithFirst(bound), key, first == bound)
						assertRangeAnswers(
							t, directed.parts.KeysFromFirst(bound), key, first >= bound)
						assertRangeAnswers(
							t, directed.parts.KeysThroughFirst(bound), key, first <= bound)
						assertRangeAnswers(
							t, directed.parts.KeysBeforeFirst(bound), key, first < bound)
					}
				}
			}
		})
	}
}

func TestKeysBeforeTheSmallestFirstValueHoldNoKey(t *testing.T) {
	parts := vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)
	keys := parts.KeysBeforeFirst(math.MinInt64)

	for _, first := range []int64{math.MinInt64, 0, math.MaxInt64} {
		assertRangeAnswers(t, keys, parts.Key(first, "").Bytes(), false)
	}
}

func assertRangeAnswers(t *testing.T, keys vault.KeyRange, key []byte, want bool) {
	t.Helper()

	if got := holdsKey(keys, key); got != want {
		t.Fatalf("range holds %x = %v, want %v", key, got, want)
	}
}

func holdsKey(keys vault.KeyRange, key []byte) bool {
	firstIncluded, firstExcluded := keys.Bounds()
	if bytes.Compare(key, firstIncluded) < 0 {
		return false
	}

	return firstExcluded == nil || bytes.Compare(key, firstExcluded) < 0
}

func TestKeysAfterAKeyHoldOnlyTheKeysOfTheRangePastIt(t *testing.T) {
	parts := vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)
	texts := []string{"", "a", "b", "\xff\xff"}
	for _, bound := range firstValuesSpanningTheEncodings {
		for _, afterFirst := range firstValuesSpanningTheEncodings {
			for _, afterSecond := range texts {
				keys := parts.KeysThroughFirst(bound).After(parts.Key(afterFirst, afterSecond))
				for _, first := range firstValuesSpanningTheEncodings {
					for _, second := range texts {
						isAfter := first > afterFirst || first == afterFirst && second > afterSecond

						assertRangeAnswers(
							t, keys, parts.Key(first, second).Bytes(), isAfter && first <= bound)
					}
				}
			}
		}
	}
}

func TestKeysAfterAKeyPastTheRangeHoldNoKeyAndStayInOrder(t *testing.T) {
	parts := vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)
	keys := parts.KeysThroughFirst(0).After(parts.Key(math.MaxInt64, "\xff\xff"))

	firstIncluded, firstExcluded := keys.Bounds()
	if bytes.Compare(firstIncluded, firstExcluded) > 0 {
		t.Fatalf("bounds %x, %x are inverted", firstIncluded, firstExcluded)
	}
	for _, first := range firstValuesSpanningTheEncodings {
		assertRangeAnswers(t, keys, parts.Key(first, "").Bytes(), false)
	}
}

func TestEveryKeyAfterAKeyHoldsEveryLaterKey(t *testing.T) {
	parts := vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)
	keys := vault.EveryKey().After(parts.Key(1, "a"))

	for _, first := range firstValuesSpanningTheEncodings {
		for _, second := range []string{"", "a", "b"} {
			isAfter := first > 1 || first == 1 && second > "a"

			assertRangeAnswers(t, keys, parts.Key(first, second).Bytes(), isAfter)
		}
	}
}
