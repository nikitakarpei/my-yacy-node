package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestPairKeyRoundTripsBothPositions(t *testing.T) {
	parts := vault.PairKey(vault.TextKeyPart, vault.TextKeyPartDescending)

	for _, first := range orderedTexts() {
		for _, second := range orderedTexts() {
			decodedFirst, decodedSecond, err := parts.PartsOf(parts.Key(first, second).Bytes())
			if err != nil {
				t.Fatalf("PartsOf(%q, %q) failed: %v", first, second, err)
			}
			if decodedFirst != first || decodedSecond != second {
				t.Fatalf(
					"PartsOf = %q, %q, want %q, %q",
					decodedFirst,
					decodedSecond,
					first,
					second,
				)
			}
		}
	}
}

func TestPairKeyRangesOfTheFirstPositionSplitAtThatFirst(t *testing.T) {
	parts := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart)
	firsts := orderedTexts()
	bound := firsts[len(firsts)/2]

	for _, first := range firsts {
		for _, second := range orderedTexts() {
			key := parts.Key(first, second).Bytes()

			assertRangeAnswers(t, parts.KeysWithFirst(bound), key, first == bound)
			assertRangeAnswers(t, parts.KeysFromFirst(bound), key, first >= bound)
			assertRangeAnswers(t, parts.KeysThroughFirst(bound), key, first <= bound)
			assertRangeAnswers(t, parts.KeysBeforeFirst(bound), key, first < bound)
		}
	}
}

func TestPairKeysSortByFirstThenSecond(t *testing.T) {
	parts := vault.PairKey(vault.TextKeyPart, vault.IntegerKeyPart)

	ordered := make([]vault.Key, 0, len(orderedTexts())*len(orderedIntegers()))
	for _, first := range orderedTexts() {
		for _, second := range orderedIntegers() {
			ordered = append(ordered, parts.Key(first, second))
		}
	}

	assertKeysSortAscending(t, func(key vault.Key) vault.Key { return key }, ordered)
}

func TestPairKeyPartsRejectsAKeyOfAnotherPartList(t *testing.T) {
	parts := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart)
	foreign := vault.SingleKey(vault.TextKeyPart).Key("only one part")

	if _, _, err := parts.PartsOf(foreign.Bytes()); err == nil {
		t.Fatal("PartsOf accepted a single-part key")
	}
}

func TestPairKeyLayoutForRoundTripsTheDomainValue(t *testing.T) {
	layout := vault.PairKey(vault.TextKeyPart, vault.IntegerKeyPart).KeyLayoutFor(
		func(word countedWord) (string, int64) { return word.text, word.count },
		func(text string, count int64) countedWord {
			return countedWord{text: text, count: count}
		},
	)

	for _, text := range orderedTexts() {
		for _, count := range orderedIntegers() {
			word := countedWord{text: text, count: count}

			decoded, err := layout.Decode(layout.Encode(word).Bytes())
			if err != nil {
				t.Fatalf("Decode(%q, %d) failed: %v", text, count, err)
			}
			if decoded != word {
				t.Fatalf("Decode = %+v, want %+v", decoded, word)
			}
		}
	}
}
