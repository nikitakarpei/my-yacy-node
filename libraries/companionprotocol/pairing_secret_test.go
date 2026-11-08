package companionprotocol_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachSecretIsNew(t *testing.T) {
	first := companionprotocol.RandomSecret()
	if second := companionprotocol.RandomSecret(); first == second {
		t.Error("two secrets are the same")
	}
}

func TestTheChallengeDependsOnTheSecretAndTheBrokerKey(t *testing.T) {
	challenge := companionprotocol.ChallengeFrom("secret-a", "key-a")

	if !bytes.Equal(challenge, companionprotocol.ChallengeFrom("secret-a", "key-a")) {
		t.Error("the same secret and key give another challenge")
	}
	if bytes.Equal(challenge, companionprotocol.ChallengeFrom("secret-b", "key-a")) {
		t.Error("another secret gives the same challenge")
	}
	if bytes.Equal(challenge, companionprotocol.ChallengeFrom("secret-a", "key-b")) {
		t.Error("another broker key gives the same challenge")
	}
}

func TestAReplyInThePairingInboxNamesTheSecret(t *testing.T) {
	secret := companionprotocol.RandomSecret()
	reply := companionprotocol.PairingInboxPrefixFor(secret) + ".token"

	if got := companionprotocol.SecretIDFrom(reply); got != companionprotocol.SecretIDOf(
		secret,
	) {
		t.Errorf("secret id = %q, want %q", got, companionprotocol.SecretIDOf(secret))
	}
}

func TestAReplyOutsideThePairingInboxNamesNoSecret(t *testing.T) {
	for _, reply := range []string{"_INBOX.abc.def", "pairing.inbox.only-id", ""} {
		if got := companionprotocol.SecretIDFrom(reply); got != "" {
			t.Errorf("secret id of %q = %q, want none", reply, got)
		}
	}
}
