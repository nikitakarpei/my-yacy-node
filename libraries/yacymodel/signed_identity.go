package yacymodel

import "time"

type SignedIdentity struct {
	Identity PeerIdentity
	IssuedAt time.Time
}

func IdentityOf(signed Optional[SignedIdentity]) Optional[PeerIdentity] {
	signedIdentity, isSigned := signed.Get()
	if !isSigned {
		return None[PeerIdentity]()
	}

	return Some(signedIdentity.Identity)
}
