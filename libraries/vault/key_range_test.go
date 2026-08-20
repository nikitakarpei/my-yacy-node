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
		layout    vault.PairKeyLayout[int64, string]
	}{
		{"Ascending", vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)},
		{"Descending", vault.PairKey(vault.IntegerKeyPartDescending, vault.TextKeyPart)},
	} {
		t.Run(directed.direction, func(t *testing.T) {
			for _, bound := range firstValuesSpanningTheEncodings {
				for _, first := range firstValuesSpanningTheEncodings {
					for _, second := range []string{"", "\xff\xff"} {
						key := directed.layout.Key(first, second).Bytes()

						assertRangeAnswers(
							t, directed.layout.KeysWithFirst(bound), key, first == bound)
						assertRangeAnswers(
							t, directed.layout.KeysFromFirst(bound), key, first >= bound)
						assertRangeAnswers(
							t, directed.layout.KeysThroughFirst(bound), key, first <= bound)
						assertRangeAnswers(
							t, directed.layout.KeysBeforeFirst(bound), key, first < bound)
					}
				}
			}
		})
	}
}

func TestKeysBeforeTheSmallestFirstValueHoldNoKey(t *testing.T) {
	layout := vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart)
	keys := layout.KeysBeforeFirst(math.MinInt64)

	for _, first := range []int64{math.MinInt64, 0, math.MaxInt64} {
		assertRangeAnswers(t, keys, layout.Key(first, "").Bytes(), false)
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
