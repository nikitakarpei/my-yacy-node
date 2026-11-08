package companionprotocol

import "strings"

const (
	pairingInboxPrefix = "pairing.inbox."

	PairingRequestSubject = "pairing.request"
)

func PairingInboxPrefixFor(secret Secret) string {
	return pairingInboxPrefix + string(SecretIDOf(secret))
}

func PairingInboxSubjectsFor(secret Secret) string {
	return PairingInboxPrefixFor(secret) + ".>"
}

func SecretIDFrom(replySubject string) SecretID {
	rest, inPairingInbox := strings.CutPrefix(replySubject, pairingInboxPrefix)
	secretID, _, found := strings.Cut(rest, ".")
	if !inPairingInbox || !found {
		return ""
	}
	return SecretID(secretID)
}
