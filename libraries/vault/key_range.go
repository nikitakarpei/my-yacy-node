package vault

import "bytes"

type KeyRange struct {
	firstIncluded []byte
	firstExcluded []byte
}

func EveryKey() KeyRange {
	return KeyRange{}
}

func (keys KeyRange) Bounds() (firstIncluded, firstExcluded []byte) {
	return bytes.Clone(keys.firstIncluded), bytes.Clone(keys.firstExcluded)
}

func (keys KeyRange) withPrefix(prefix []byte) KeyRange {
	prefixed := KeyRange{
		firstIncluded: append(bytes.Clone(prefix), keys.firstIncluded...),
		firstExcluded: successorOf(prefix),
	}
	if keys.firstExcluded != nil {
		prefixed.firstExcluded = append(bytes.Clone(prefix), keys.firstExcluded...)
	}

	return prefixed
}

// No codec emits an all-0xFF encoding, so the nil successor never stands for a real key.
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
