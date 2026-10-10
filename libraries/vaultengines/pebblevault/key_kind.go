package pebblevault

import "github.com/nikitakarpei/yacy-rwi-node/vault"

type KeyKind string

const (
	KindRecords KeyKind = "records"
	KindTallies KeyKind = "tallies"
	KindAmounts KeyKind = "amounts"
)

func bucketAndKindOf(absoluteKey []byte) (vault.Name, KeyKind) {
	switch absoluteKey[0] {
	case tallyRegionPrefix:
		return tallyBucketFrom(absoluteKey), KindTallies
	case amountsRegionPrefix:
		return bucketNameFrom(absoluteKey), KindAmounts
	}

	return bucketNameFrom(absoluteKey), KindRecords
}
