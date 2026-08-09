package vaultkey_test

import (
	"bytes"
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

func TestTriplePrefixesAreBytePrefixesOfTheFullKey(t *testing.T) {
	layout := vaultkey.Triple(vaultkey.Text, vaultkey.Text, vaultkey.Text)

	for _, first := range orderedTexts() {
		for _, second := range orderedTexts() {
			firstPrefix := layout.First(first).Bytes()
			firstTwoPrefix := layout.FirstTwo(first, second).Bytes()

			if !bytes.HasPrefix(firstTwoPrefix, firstPrefix) {
				t.Fatalf("FirstTwo(%q, %q) = %x does not start with First(%q) = %x",
					first, second, firstTwoPrefix, first, firstPrefix)
			}

			for _, third := range orderedTexts() {
				full := layout.Key(first, second, third).Bytes()
				if !bytes.HasPrefix(full, firstTwoPrefix) {
					t.Fatalf("Key(%q, %q, %q) = %x does not start with %x",
						first, second, third, full, firstTwoPrefix)
				}
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
