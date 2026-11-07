package companionprotocol_test

import (
	"bytes"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachSecretIsNew(t *testing.T) {
	first := companionprotocol.NewSecret()
	if second := companionprotocol.NewSecret(); first == second {
		t.Error("two secrets are the same")
	}
}

func TestTheChallengeDependsOnTheSecretAndTheBrokerKey(t *testing.T) {
	challenge := companionprotocol.ChallengeFor("secret-a", "key-a")

	if !bytes.Equal(challenge, companionprotocol.ChallengeFor("secret-a", "key-a")) {
		t.Error("the same secret and key give another challenge")
	}
	if bytes.Equal(challenge, companionprotocol.ChallengeFor("secret-b", "key-a")) {
		t.Error("another secret gives the same challenge")
	}
	if bytes.Equal(challenge, companionprotocol.ChallengeFor("secret-a", "key-b")) {
		t.Error("another broker key gives the same challenge")
	}
}

func TestAReplyInThePairingInboxNamesTheSecret(t *testing.T) {
	secret := companionprotocol.NewSecret()
	reply := companionprotocol.PairingInboxPrefixFor(secret) + ".token"

	if got := companionprotocol.SecretIDFromReply(reply); got != companionprotocol.SecretIDOf(
		secret,
	) {
		t.Errorf("secret id = %q, want %q", got, companionprotocol.SecretIDOf(secret))
	}
}

func TestAReplyOutsideThePairingInboxNamesNoSecret(t *testing.T) {
	for _, reply := range []string{"_INBOX.abc.def", "pairing.replies.only-id", ""} {
		if got := companionprotocol.SecretIDFromReply(reply); got != "" {
			t.Errorf("secret id of %q = %q, want none", reply, got)
		}
	}
}
