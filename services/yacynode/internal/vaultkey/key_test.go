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

	decoded, err := layout.Parts(key.Bytes())
	if err != nil {
		t.Fatalf("Parts failed: %v", err)
	}
	if decoded != "word" {
		t.Fatalf("Parts = %q, want %q", decoded, "word")
	}
}

func TestPartsRejectsBytesNoLayoutProduced(t *testing.T) {
	layout := vaultkey.Single(vaultkey.Text)

	if _, err := layout.Parts([]byte{0xFF, 0xFE}); err == nil {
		t.Fatal("Parts accepted bytes no layout produced")
	}
}
