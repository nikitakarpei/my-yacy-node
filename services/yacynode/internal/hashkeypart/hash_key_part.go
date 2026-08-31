// Package hashkeypart carries the vault key parts for the yacymodel hashes, so
// every vault that keys on a hash shares one byte form and one parser.
package hashkeypart

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var Hash = vault.BytesKeyPartFrom(yacymodel.Hash.Bytes, yacymodel.ParseHashBytes)

var URLHash = vault.BytesKeyPartFrom(yacymodel.URLHash.Bytes, yacymodel.ParseURLHashBytes)
