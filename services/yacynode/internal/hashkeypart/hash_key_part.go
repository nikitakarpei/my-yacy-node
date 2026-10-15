// Package hashkeypart carries the vault key parts for the yacymodel hashes, so
// every vault that keys on a hash shares one byte form and one parser.
package hashkeypart

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var Hash = vault.BytesKeyPartFrom(yacymodel.Hash.AppendBytes, hashFrom)

var URLHash = vault.BytesKeyPartFrom(yacymodel.URLHash.AppendBytes, urlHashFrom)

func hashFrom(stored vault.StoredBytes) (yacymodel.Hash, error) {
	var raw [yacymodel.HashByteLength]byte

	return yacymodel.ParseHashBytes(stored.AppendTo(raw[:0]))
}

func urlHashFrom(stored vault.StoredBytes) (yacymodel.URLHash, error) {
	var raw [yacymodel.HashByteLength]byte

	return yacymodel.ParseURLHashBytes(stored.AppendTo(raw[:0]))
}
