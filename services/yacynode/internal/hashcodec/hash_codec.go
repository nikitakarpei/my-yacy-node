// Package hashcodec carries the vault key codecs for the yacymodel hashes, so
// every vault that keys on a hash shares one text form and one parser.
package hashcodec

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var Hash = vault.TextKeyPartFrom(yacymodel.Hash.String, yacymodel.ParseHash)

var URLHash = vault.TextKeyPartFrom(yacymodel.URLHash.String, yacymodel.ParseURLHash)
