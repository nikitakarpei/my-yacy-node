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
