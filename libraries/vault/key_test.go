package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestAKeyDoesNotShareItsBytesWithTheCaller(t *testing.T) {
	parts := vault.SingleKey(vault.TextKeyPart)
	key := parts.Key("word")

	handedOut := key.Bytes()
	for position := range handedOut {
		handedOut[position] = 0
	}

	decoded, err := parts.PartsOf(key.Bytes())
	if err != nil {
		t.Fatalf("PartsOf failed: %v", err)
	}
	if decoded != "word" {
		t.Fatalf("PartsOf = %q, want %q", decoded, "word")
	}
}

func TestPartsOfRejectsBytesNoPartListProduced(t *testing.T) {
	parts := vault.SingleKey(vault.TextKeyPart)

	if _, err := parts.PartsOf([]byte{0xFF, 0xFE}); err == nil {
		t.Fatal("PartsOf accepted bytes no part list produced")
	}
}
