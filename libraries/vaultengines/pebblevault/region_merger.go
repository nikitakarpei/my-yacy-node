package pebblevault

import "github.com/cockroachdb/pebble/v2"

var regionMerger = &pebble.Merger{
	Name:  "pebblevault.amount_sum",
	Merge: foldFor,
}

func foldFor(key, operand []byte) (pebble.ValueMerger, error) {
	if len(key) > 0 && key[0] == memberSetsRegionPrefix {
		return memberSetFoldFrom(operand)
	}

	return amountSumFrom(operand)
}
