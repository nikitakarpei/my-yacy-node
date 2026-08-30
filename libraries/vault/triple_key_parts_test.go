package vault_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestTripleKeyRoundTripsAllThreePositions(t *testing.T) {
	parts := vault.TripleKey(vault.TimeKeyPart, vault.TextKeyPart, vault.IntegerKeyPart)

	for _, instant := range orderedInstants() {
		for _, text := range orderedTexts() {
			for _, number := range orderedIntegers() {
				key := parts.Key(instant, text, number)

				decodedInstant, decodedText, decodedNumber, err := parts.PartsOf(key.Bytes())
				if err != nil {
					t.Fatalf("PartsOf(%s, %q, %d) failed: %v", instant, text, number, err)
				}
				if !decodedInstant.Equal(instant) || decodedText != text ||
					decodedNumber != number {
					t.Fatalf("PartsOf = %s, %q, %d, want %s, %q, %d",
						decodedInstant, decodedText, decodedNumber, instant, text, number)
				}
			}
		}
	}
}

func TestTripleKeyRangesOfTheFirstPositionSplitAtThatFirst(t *testing.T) {
	parts := vault.TripleKey(vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)
	firsts := orderedTexts()
	bound := firsts[len(firsts)/2]

	for _, first := range firsts {
		for _, second := range orderedTexts() {
			for _, third := range orderedTexts() {
				key := parts.Key(first, second, third).Bytes()

				assertRangeAnswers(t, parts.KeysWithFirst(bound), key, first == bound)
				assertRangeAnswers(t, parts.KeysFromFirst(bound), key, first >= bound)
				assertRangeAnswers(t, parts.KeysThroughFirst(bound), key, first <= bound)
				assertRangeAnswers(t, parts.KeysBeforeFirst(bound), key, first < bound)
			}
		}
	}
}

func TestTripleKeyPartsRejectsAKeyOfAnotherPartList(t *testing.T) {
	parts := vault.TripleKey(vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)
	foreign := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).Key("one", "two")

	if _, _, _, err := parts.PartsOf(foreign.Bytes()); err == nil {
		t.Fatal("PartsOf accepted a two-part key")
	}
}

func TestTripleKeyLayoutForRoundTripsTheDomainValue(t *testing.T) {
	layout := vault.TripleKey(
		vault.TimeKeyPart,
		vault.TextKeyPart,
		vault.IntegerKeyPart,
	).KeyLayoutFor(
		func(word countedWord) (time.Time, string, int64) {
			return word.seenAt, word.text, word.count
		},
		func(seenAt time.Time, text string, count int64) countedWord {
			return countedWord{seenAt: seenAt, text: text, count: count}
		},
	)

	for _, instant := range orderedInstants() {
		word := countedWord{seenAt: instant, text: "word", count: 7}

		decoded, err := layout.Decode(layout.Encode(word).Bytes())
		if err != nil {
			t.Fatalf("Decode(%s) failed: %v", instant, err)
		}
		if !decoded.seenAt.Equal(instant) || decoded.text != word.text ||
			decoded.count != word.count {
			t.Fatalf("Decode = %+v, want %+v", decoded, word)
		}
	}
}
