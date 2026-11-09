package companionprotocol_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestThePairingRequestSubjectKeepsItsWireForm(t *testing.T) {
	if want := "pairing.request"; companionprotocol.PairingRequestSubject != want {
		t.Errorf("subject = %q, want %q", companionprotocol.PairingRequestSubject, want)
	}
}

func TestASubjectInThePairingInboxNamesTheSecret(t *testing.T) {
	secret := companionprotocol.RandomSecret()
	inboxSubject := companionprotocol.PairingInboxPrefixFor(secret) + ".token"

	if got := companionprotocol.SecretIDIn(
		inboxSubject,
	); got != companionprotocol.SecretIDFrom(
		secret,
	) {
		t.Errorf("secret id = %q, want %q", got, companionprotocol.SecretIDFrom(secret))
	}
}

func TestASubjectOutsideThePairingInboxesNamesNoSecret(t *testing.T) {
	for _, subject := range []string{"_INBOX.abc.def", "pairing.inbox.only-id", ""} {
		if got := companionprotocol.SecretIDIn(subject); got != "" {
			t.Errorf("secret id in %q = %q, want none", subject, got)
		}
	}
}

func TestThePairingInboxSubjectsCoverTheInboxesOfTheirSecretOnly(t *testing.T) {
	secret := companionprotocol.RandomSecret()
	inboxSubject := companionprotocol.PairingInboxPrefixFor(secret) + ".token"

	if !covers(companionprotocol.PairingInboxSubjectsFor(secret), inboxSubject) {
		t.Errorf("the inbox subjects of the secret do not cover %q", inboxSubject)
	}
	otherSecret := companionprotocol.RandomSecret()
	if covers(companionprotocol.PairingInboxSubjectsFor(otherSecret), inboxSubject) {
		t.Errorf("the inbox subjects of another secret cover %q", inboxSubject)
	}
}
