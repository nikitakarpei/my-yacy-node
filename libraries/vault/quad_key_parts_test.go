package vault_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestQuadKeyRoundTripsAllFourPositions(t *testing.T) {
	parts := vault.QuadKey(
		vault.TimeKeyPart, vault.TextKeyPart, vault.IntegerKeyPart, vault.TextKeyPart)

	for _, instant := range orderedInstants() {
		for _, text := range orderedTexts() {
			for _, number := range orderedIntegers() {
				key := parts.Key(instant, text, number, text)

				decodedInstant, decodedText, decodedNumber, decodedLast, err := parts.PartsOf(
					key.Bytes())
				if err != nil {
					t.Fatalf("PartsOf(%s, %q, %d, %q) failed: %v", instant, text, number, text, err)
				}
				if !decodedInstant.Equal(instant) || decodedText != text ||
					decodedNumber != number || decodedLast != text {
					t.Fatalf("PartsOf = %s, %q, %d, %q, want %s, %q, %d, %q",
						decodedInstant, decodedText, decodedNumber, decodedLast,
						instant, text, number, text)
				}
			}
		}
	}
}

func TestQuadKeyRangeOfOneFirstHoldsOnlyThatFirst(t *testing.T) {
	parts := vault.QuadKey(
		vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)
	firsts := orderedTexts()
	bound := firsts[len(firsts)/2]

	for _, first := range firsts {
		for _, rest := range orderedTexts() {
			key := parts.Key(first, rest, rest, rest).Bytes()

			assertRangeAnswers(t, parts.KeysWithFirst(bound), key, first == bound)
		}
	}
}

func TestQuadKeyRangeOfOneFirstThroughASecondHoldsOnlyThoseKeys(t *testing.T) {
	for _, directed := range []struct {
		direction string
		parts     vault.QuadKeyParts[string, int64, string, string]
	}{
		{"Ascending", vault.QuadKey(
			vault.TextKeyPart, vault.IntegerKeyPart, vault.TextKeyPart, vault.TextKeyPart)},
		{"Descending", vault.QuadKey(
			vault.TextKeyPart, vault.IntegerKeyPartDescending, vault.TextKeyPart, vault.TextKeyPart)},
	} {
		t.Run(directed.direction, func(t *testing.T) {
			firsts, seconds := orderedTexts(), orderedIntegers()
			firstBound, secondBound := firsts[len(firsts)/2], seconds[len(seconds)/2]
			keys := directed.parts.KeysWithFirstThroughSecond(firstBound, secondBound)

			for _, first := range firsts {
				for _, second := range seconds {
					for _, rest := range []string{"", "\xff\xff"} {
						key := directed.parts.Key(first, second, rest, rest).Bytes()

						assertRangeAnswers(
							t, keys, key, first == firstBound && second <= secondBound)
					}
				}
			}
		})
	}
}

func TestQuadKeyPartsRejectsAKeyOfAnotherPartList(t *testing.T) {
	parts := vault.QuadKey(
		vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)
	foreign := vault.TripleKey(
		vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart).Key("one", "two", "three")

	if _, _, _, _, err := parts.PartsOf(foreign.Bytes()); err == nil {
		t.Fatal("PartsOf accepted a three-part key")
	}
}

func TestQuadKeyLayoutForRoundTripsTheDomainValue(t *testing.T) {
	layout := vault.QuadKey(
		vault.TimeKeyPart,
		vault.TextKeyPart,
		vault.IntegerKeyPart,
		vault.TextKeyPart,
	).KeyLayoutFor(
		func(word countedWord) (time.Time, string, int64, string) {
			return word.seenAt, word.text, word.count, word.text
		},
		func(seenAt time.Time, text string, count int64, _ string) countedWord {
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
