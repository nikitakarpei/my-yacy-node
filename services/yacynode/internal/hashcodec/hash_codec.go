// Package hashcodec carries the vault key codecs for the yacymodel hashes, so
// every vault that keys on a hash shares one text form and one parser.
package hashcodec

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var Hash = vaultkey.TextAs(yacymodel.Hash.String, yacymodel.ParseHash)

var URLHash = vaultkey.TextAs(yacymodel.URLHash.String, yacymodel.ParseURLHash)
