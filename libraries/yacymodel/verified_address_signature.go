package yacymodel

import "time"

type VerifiedAddressSignature struct {
	Identity PeerIdentity
	IssuedAt time.Time
}
