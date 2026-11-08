package companionprotocol_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachSecretIsNew(t *testing.T) {
	first := companionprotocol.RandomSecret()
	if second := companionprotocol.RandomSecret(); first == second {
		t.Error("two secrets are the same")
	}
}

func TestEachSecretGivesItsOwnSecretID(t *testing.T) {
	secret := companionprotocol.RandomSecret()
	sameSecret := companionprotocol.Secret(string(secret))

	if companionprotocol.SecretIDFrom(secret) != companionprotocol.SecretIDFrom(sameSecret) {
		t.Error("one secret gives two secret ids")
	}
	otherSecret := companionprotocol.RandomSecret()
	if companionprotocol.SecretIDFrom(secret) == companionprotocol.SecretIDFrom(otherSecret) {
		t.Error("two secrets give the same secret id")
	}
}
