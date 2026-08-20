package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestAKeyDoesNotShareItsBytesWithTheCaller(t *testing.T) {
	layout := vault.SingleKey(vault.TextKeyPart)
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
	layout := vault.SingleKey(vault.TextKeyPart)

	if _, err := layout.Parts([]byte{0xFF, 0xFE}); err == nil {
		t.Fatal("Parts accepted bytes no layout produced")
	}
}
