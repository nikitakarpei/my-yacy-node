package pebblevault

import (
	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/internal/membersetfold"
)

var regionMerger = &pebble.Merger{
	Name:  "pebblevault.amount_sum",
	Merge: valueMergerFor,
}

func valueMergerFor(key, operand []byte) (pebble.ValueMerger, error) {
	if isMemberSetKey(key) {
		return membersetfold.From(operand)
	}

	return amountSumFrom(operand)
}
