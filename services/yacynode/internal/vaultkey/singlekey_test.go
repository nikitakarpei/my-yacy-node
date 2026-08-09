package vaultkey_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestSinglePartsRejectsAKeyOfAnotherLayout(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)
	foreign := vaultkey.Pair(vaultkey.Text, vaultkey.Text).Key("one", "two")

	if _, err := layout.Parts(foreign); err == nil {
		t.Fatal("Parts accepted a two-part key")
	}
}
