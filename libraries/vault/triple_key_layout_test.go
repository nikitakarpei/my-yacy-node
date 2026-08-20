package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestTripleKeyRoundTripsAllThreePositions(t *testing.T) {
	layout := vault.TripleKey(vault.TimeKeyPart, vault.TextKeyPart, vault.IntegerKeyPart)

	for _, instant := range orderedInstants() {
		for _, text := range orderedTexts() {
			for _, number := range orderedIntegers() {
				key := layout.Key(instant, text, number)

				decodedInstant, decodedText, decodedNumber, err := layout.Parts(key.Bytes())
				if err != nil {
					t.Fatalf("Parts(%s, %q, %d) failed: %v", instant, text, number, err)
				}
				if !decodedInstant.Equal(instant) || decodedText != text ||
					decodedNumber != number {
					t.Fatalf("Parts = %s, %q, %d, want %s, %q, %d",
						decodedInstant, decodedText, decodedNumber, instant, text, number)
				}
			}
		}
	}
}

func TestTripleKeyRangesOfTheFirstPositionSplitAtThatFirst(t *testing.T) {
	layout := vault.TripleKey(vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)
	firsts := orderedTexts()
	bound := firsts[len(firsts)/2]

	for _, first := range firsts {
		for _, second := range orderedTexts() {
			for _, third := range orderedTexts() {
				key := layout.Key(first, second, third).Bytes()

				assertRangeAnswers(t, layout.KeysWithFirst(bound), key, first == bound)
				assertRangeAnswers(t, layout.KeysFromFirst(bound), key, first >= bound)
				assertRangeAnswers(t, layout.KeysThroughFirst(bound), key, first <= bound)
				assertRangeAnswers(t, layout.KeysBeforeFirst(bound), key, first < bound)
			}
		}
	}
}

func TestTripleKeyPartsRejectsAKeyOfAnotherLayout(t *testing.T) {
	layout := vault.TripleKey(vault.TextKeyPart, vault.TextKeyPart, vault.TextKeyPart)
	foreign := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).Key("one", "two")

	if _, _, _, err := layout.Parts(foreign.Bytes()); err == nil {
		t.Fatal("Parts accepted a two-part key")
	}
}
