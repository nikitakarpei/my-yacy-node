package postingidentity

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

var identityKeyParts = vault.PairKey(hashkeypart.Hash, hashkeypart.URLHash)

var KeyLayout = identityKeyParts.KeyLayoutFor(
	func(identity Identity) (yacymodel.Hash, yacymodel.URLHash) {
		return identity.Word, identity.URL
	},
	IdentityOf,
)
