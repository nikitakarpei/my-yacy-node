package postingidentity

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
)

var identityKeyLayout = vault.PairKey(hashcodec.Hash, hashcodec.URLHash)

var KeyCodec = identityKeyLayout.KeyCodecFor(
	func(identity Identity) (yacymodel.Hash, yacymodel.URLHash) {
		return identity.Word, identity.URL
	},
	IdentityOf,
)
