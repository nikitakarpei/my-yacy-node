package pebblevault

import "github.com/cockroachdb/pebble/v2"

var regionMerger = &pebble.Merger{
	Name:  "pebblevault.amount_sum",
	Merge: valueMergerFor,
}

func valueMergerFor(key, operand []byte) (pebble.ValueMerger, error) {
	if isMemberSetKey(key) {
		return memberSetFoldFrom(operand)
	}

	return amountSumFrom(operand)
}
