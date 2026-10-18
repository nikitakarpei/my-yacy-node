package vault_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func TestValuesThatRepeatAValueAreRefused(t *testing.T) {
	if _, err := vault.FixedDictionaryFrom(
		vault.TextKeyPart,
		[]string{"the", "a", "the"},
	); err == nil {
		t.Fatal("FixedDictionaryFrom with a repeated value succeeded, want refusal")
	}
}
