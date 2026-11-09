package companionprotocol

import "strings"

const (
	pairingInboxesPrefix = "pairing.inbox."

	PairingRequestSubject = "pairing.request"
)

func PairingInboxPrefixFor(secret Secret) string {
	return pairingInboxesPrefix + string(SecretIDFrom(secret))
}

func PairingInboxSubjectsFor(secret Secret) string {
	return PairingInboxPrefixFor(secret) + ".>"
}

func SecretIDIn(subject string) SecretID {
	rest, inPairingInboxes := strings.CutPrefix(subject, pairingInboxesPrefix)
	secretID, _, found := strings.Cut(rest, ".")
	if !inPairingInboxes || !found {
		return ""
	}
	return SecretID(secretID)
}
