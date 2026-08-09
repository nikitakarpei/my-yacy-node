package vaultkey_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestSingleRangesOfTheOnlyPositionSplitAtThatPosition(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)
	texts := orderedTexts()
	bound := texts[len(texts)/2]

	for _, text := range texts {
		key := layout.Key(text).Bytes()

		assertRangeAnswers(t, layout.KeysWithFirst(bound), key, text == bound)
		assertRangeAnswers(t, layout.KeysThroughFirst(bound), key, text <= bound)
		assertRangeAnswers(t, layout.KeysBeforeFirst(bound), key, text < bound)
	}
}

func TestSinglePartsRejectsAKeyOfAnotherLayout(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)
	foreign := vaultkey.Pair(vaultkey.Text, vaultkey.Text).Key("one", "two")

	if _, err := layout.Parts(foreign); err == nil {
		t.Fatal("Parts accepted a two-part key")
	}
}
