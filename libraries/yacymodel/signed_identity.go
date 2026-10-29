package yacymodel

import "time"

type SignedIdentity struct {
	Identity PeerIdentity
	IssuedAt time.Time
}
