package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestSingleKeyRangesOfTheOnlyPositionSplitAtThatPosition(t *testing.T) {
	layout := vault.SingleKey(vault.TextKeyPart)
	texts := orderedTexts()
	bound := texts[len(texts)/2]

	for _, text := range texts {
		key := layout.Key(text).Bytes()

		assertRangeAnswers(t, layout.KeysWithFirst(bound), key, text == bound)
		assertRangeAnswers(t, layout.KeysFromFirst(bound), key, text >= bound)
		assertRangeAnswers(t, layout.KeysThroughFirst(bound), key, text <= bound)
		assertRangeAnswers(t, layout.KeysBeforeFirst(bound), key, text < bound)
	}
}

func TestSingleKeyPartsRejectsAKeyOfAnotherLayout(t *testing.T) {
	layout := vault.SingleKey(vault.TextKeyPart)
	foreign := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).Key("one", "two")

	if _, err := layout.Parts(foreign.Bytes()); err == nil {
		t.Fatal("Parts accepted a two-part key")
	}
}

func TestSingleKeyCodecRoundTripsThePartItself(t *testing.T) {
	codec := vault.SingleKey(vault.TextKeyPart).KeyCodec()

	for _, text := range orderedTexts() {
		decoded, err := codec.Decode(codec.Encode(text).Bytes())
		if err != nil {
			t.Fatalf("Decode(%q) failed: %v", text, err)
		}
		if decoded != text {
			t.Fatalf("Decode = %q, want %q", decoded, text)
		}
	}
}

func TestSingleKeyCodecForRoundTripsTheDomainValue(t *testing.T) {
	codec := vault.SingleKey(vault.IntegerKeyPart).KeyCodecFor(
		func(word countedWord) int64 { return word.count },
		func(count int64) countedWord { return countedWord{count: count} },
	)

	for _, count := range orderedIntegers() {
		decoded, err := codec.Decode(codec.Encode(countedWord{count: count}).Bytes())
		if err != nil {
			t.Fatalf("Decode(%d) failed: %v", count, err)
		}
		if decoded.count != count {
			t.Fatalf("Decode = %d, want %d", decoded.count, count)
		}
	}
}

func TestSingleKeyCodecForReportsAKeyOfAnotherLayout(t *testing.T) {
	codec := vault.SingleKey(vault.TextKeyPart).KeyCodec()
	foreign := vault.PairKey(vault.TextKeyPart, vault.TextKeyPart).Key("one", "two")

	if _, err := codec.Decode(foreign.Bytes()); err == nil {
		t.Fatal("Decode accepted a two-part key")
	}
}
