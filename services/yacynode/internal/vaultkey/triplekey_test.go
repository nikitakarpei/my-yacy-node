package vaultkey_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestTripleRoundTripsAllThreePositions(t *testing.T) {
	layout := vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Integer)

	for _, instant := range orderedInstants() {
		for _, text := range orderedTexts() {
			for _, number := range orderedIntegers() {
				key := layout.Key(instant, text, number)

				decodedInstant, decodedText, decodedNumber, err := layout.Parts(key)
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

func TestTripleRangesOfTheFirstPositionSplitAtThatFirst(t *testing.T) {
	layout := vaultkey.Triple(vaultkey.Text, vaultkey.Text, vaultkey.Text)
	firsts := orderedTexts()
	bound := firsts[len(firsts)/2]

	for _, first := range firsts {
		for _, second := range orderedTexts() {
			for _, third := range orderedTexts() {
				key := layout.Key(first, second, third).Bytes()

				assertRangeAnswers(t, layout.KeysWithFirst(bound), key, first == bound)
				assertRangeAnswers(t, layout.KeysThroughFirst(bound), key, first <= bound)
				assertRangeAnswers(t, layout.KeysBeforeFirst(bound), key, first < bound)
			}
		}
	}
}

func TestTriplePartsRejectsAKeyOfAnotherLayout(t *testing.T) {
	layout := vaultkey.Triple(vaultkey.Text, vaultkey.Text, vaultkey.Text)
	foreign := vaultkey.Pair(vaultkey.Text, vaultkey.Text).Key("one", "two")

	if _, _, _, err := layout.Parts(foreign); err == nil {
		t.Fatal("Parts accepted a two-part key")
	}
}
