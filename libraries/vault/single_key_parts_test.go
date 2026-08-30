package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestSingleKeyRangesOfTheOnlyPositionSplitAtThatPosition(t *testing.T) {
	parts := vault.SingleKey(vault.TextKeyPart)
	texts := orderedTexts()
	bound := texts[len(texts)/2]

	for _, text := range texts {
		key := parts.Key(text).Bytes()

		assertRangeAnswers(t, parts.KeysWithFirst(bound), key, text == bound)
		assertRangeAnswers(t, parts.KeysFromFirst(bound), key, text >= bound)
		assertRangeAnswers(t, parts.KeysThroughFirst(bound), key, text <= bound)
		assertRangeAnswers(t, parts.KeysBeforeFirst(bound), key, text < bound)
	}
}

func TestSingleKeyPartsRejectsAKeyOfAnotherPartList(t *testing.T) {
	parts := vault.SingleKey(vault.TextKeyPart)
	foreign := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).Key("one", "two")

	if _, err := parts.PartsOf(foreign.Bytes()); err == nil {
		t.Fatal("PartsOf accepted a two-part key")
	}
}

func TestSingleKeyLayoutRoundTripsThePartItself(t *testing.T) {
	layout := vault.SingleKey(vault.TextKeyPart).KeyLayout()

	for _, text := range orderedTexts() {
		decoded, err := layout.Decode(layout.Encode(text).Bytes())
		if err != nil {
			t.Fatalf("Decode(%q) failed: %v", text, err)
		}
		if decoded != text {
			t.Fatalf("Decode = %q, want %q", decoded, text)
		}
	}
}

func TestSingleKeyLayoutForRoundTripsTheDomainValue(t *testing.T) {
	layout := vault.SingleKey(vault.IntegerKeyPart).KeyLayoutFor(
		func(word countedWord) int64 { return word.count },
		func(count int64) countedWord { return countedWord{count: count} },
	)

	for _, count := range orderedIntegers() {
		decoded, err := layout.Decode(layout.Encode(countedWord{count: count}).Bytes())
		if err != nil {
			t.Fatalf("Decode(%d) failed: %v", count, err)
		}
		if decoded.count != count {
			t.Fatalf("Decode = %d, want %d", decoded.count, count)
		}
	}
}

func TestSingleKeyLayoutForReportsAKeyOfAnotherPartList(t *testing.T) {
	layout := vault.SingleKey(vault.TextKeyPart).KeyLayout()
	foreign := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).Key("one", "two")

	if _, err := layout.Decode(foreign.Bytes()); err == nil {
		t.Fatal("Decode accepted a two-part key")
	}
}
