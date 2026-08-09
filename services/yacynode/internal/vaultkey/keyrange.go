package vaultkey

import "bytes"

type KeyRange struct {
	firstIncluded []byte
	firstExcluded []byte
}

func EveryKey() KeyRange {
	return KeyRange{}
}

func KeysFrom(bound Key) KeyRange {
	return KeyRange{firstIncluded: bound.encoded}
}

func (keys KeyRange) Bounds() (firstIncluded, firstExcluded []byte) {
	return bytes.Clone(keys.firstIncluded), bytes.Clone(keys.firstExcluded)
}

func successorOf(prefix []byte) []byte {
	for position := len(prefix) - 1; position >= 0; position-- {
		if prefix[position] == 0xFF {
			continue
		}

		successor := make([]byte, position+1)
		copy(successor, prefix[:position+1])
		successor[position]++

		return successor
	}

	return nil
}
