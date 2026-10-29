package yacymodel

import "crypto/ed25519"

const tlsKeyEndorsementDomain = "yacy-tls-key/1"

type TLSKeyEndorsement struct {
	Identity  PeerIdentity
	Signature [ed25519.SignatureSize]byte
}

func (e TLSKeyEndorsement) Endorses(tlsPublicKey []byte, networkName string) bool {
	return e.Identity.verifies(
		tlsKeyClaimFrom(e.Identity.Hash(), tlsPublicKey, networkName),
		e.Signature[:],
	)
}

func tlsKeyClaimFrom(peerHash Hash, tlsPublicKey []byte, networkName string) []byte {
	claim := []byte(tlsKeyEndorsementDomain)
	claim = appendClaimField(claim, networkName)
	claim = appendClaimField(claim, peerHash.String())

	return appendClaimField(claim, string(tlsPublicKey))
}
