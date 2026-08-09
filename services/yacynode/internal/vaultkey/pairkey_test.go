package vaultkey_test

import (
	"bytes"
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

func TestPairFirstIsABytePrefixOfTheFullKey(t *testing.T) {
	layout := vaultkey.Pair(vaultkey.Text, vaultkey.Text)

	for _, first := range orderedTexts() {
		prefix := layout.First(first).Bytes()

		for _, second := range orderedTexts() {
			full := layout.Key(first, second).Bytes()
			if !bytes.HasPrefix(full, prefix) {
				t.Fatalf("Key(%q, %q) = %x does not start with First(%q) = %x",
					first, second, full, first, prefix)
			}
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
