package vaultkey_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

func TestAKeyDoesNotShareItsBytesWithTheCaller(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)
	key := layout.Key("word")

	handedOut := key.Bytes()
	for position := range handedOut {
		handedOut[position] = 0
	}

	decoded, err := layout.Parts(key)
	if err != nil {
		t.Fatalf("Parts failed: %v", err)
	}
	if decoded != "word" {
		t.Fatalf("Parts = %q, want %q", decoded, "word")
	}
}

func TestKeyFromCopiesTheBytesItIsGiven(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)
	stored := layout.Key("word").Bytes()

	restored := vaultkey.KeyFrom(stored)
	for position := range stored {
		stored[position] = 0
	}

	decoded, err := layout.Parts(restored)
	if err != nil {
		t.Fatalf("Parts failed: %v", err)
	}
	if decoded != "word" {
		t.Fatalf("Parts = %q, want %q", decoded, "word")
	}
}

func TestPartsRejectsBytesNoLayoutProduced(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)

	if _, err := layout.Parts(vaultkey.KeyFrom([]byte{0xFF, 0xFE})); err == nil {
		t.Fatal("Parts accepted bytes no layout produced")
	}
}
