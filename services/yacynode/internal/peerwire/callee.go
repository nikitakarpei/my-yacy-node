package peerwire

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Callee struct {
	hash     yacymodel.Hash
	identity yacymodel.Optional[yacymodel.PeerIdentity]
}

func IdentifiedCallee(identity yacymodel.PeerIdentity) Callee {
	return Callee{hash: identity.Hash(), identity: yacymodel.Some(identity)}
}

func UnidentifiedCallee(peerHash yacymodel.Hash) Callee {
	return Callee{hash: peerHash}
}

func (c Callee) Hash() yacymodel.Hash {
	return c.hash
}

func (c Callee) Identity() yacymodel.Optional[yacymodel.PeerIdentity] {
	return c.identity
}
