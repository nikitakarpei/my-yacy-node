package vaultkey_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestPairRoundTripsBothPositions(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.TextDescending)

	for _, first := range orderedTexts() {
		for _, second := range orderedTexts() {
			decodedFirst, decodedSecond, err := layout.Parts(layout.Key(first, second))
			if err != nil {
				t.Fatalf("Parts(%q, %q) failed: %v", first, second, err)
			}
			if decodedFirst != first || decodedSecond != second {
				t.Fatalf("Parts = %q, %q, want %q, %q", decodedFirst, decodedSecond, first, second)
			}
		}
	}
}

func TestPairRangesOfTheFirstPositionSplitAtThatFirst(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	firsts := orderedTexts()
	bound := firsts[len(firsts)/2]

	for _, first := range firsts {
		for _, second := range orderedTexts() {
			key := layout.Key(first, second).Bytes()

			assertRangeAnswers(t, layout.KeysWithFirst(bound), key, first == bound)
			assertRangeAnswers(t, layout.KeysThroughFirst(bound), key, first <= bound)
			assertRangeAnswers(t, layout.KeysBeforeFirst(bound), key, first < bound)
		}
	}
}

func TestPairKeysSortByFirstThenSecond(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Integer)

	ordered := make([]vaultkey.Key, 0, len(orderedTexts())*len(orderedIntegers()))
	for _, first := range orderedTexts() {
		for _, second := range orderedIntegers() {
			ordered = append(ordered, layout.Key(first, second))
		}
	}

	assertKeysSortAscending(t, func(key vaultkey.Key) vaultkey.Key { return key }, ordered)
}

func TestPairPartsRejectsAKeyOfAnotherLayout(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	foreign := vaultkey.Single(vaultkey.Text).Key("only one part")

	if _, _, err := layout.Parts(foreign); err == nil {
		t.Fatal("Parts accepted a single-part key")
	}
}
