package pebblevault

import (
	"bytes"
	"fmt"
	"slices"
	"sort"

	"github.com/cockroachdb/pebble/v2"
)

type sortedKeys [][]byte

func writtenKeysOf(changes *pebble.Batch) (sortedKeys, error) {
	writtenKeys := make(sortedKeys, 0, changes.Count())
	var keyBytes []byte
	var keyEnds []int

	entries := changes.Reader()
	for {
		_, key, _, found, err := entries.Next()
		if err != nil {
			return nil, fmt.Errorf("read staged changes: %w", err)
		}
		if !found {
			break
		}
		keyBytes = append(keyBytes, key...)
		keyEnds = append(keyEnds, len(keyBytes))
	}
	keyStart := 0
	for _, keyEnd := range keyEnds {
		writtenKeys = append(writtenKeys, keyBytes[keyStart:keyEnd:keyEnd])
		keyStart = keyEnd
	}
	slices.SortFunc(writtenKeys, bytes.Compare)

	return slices.CompactFunc(writtenKeys, bytes.Equal), nil
}

func (k sortedKeys) firstWithin(visited visitedRange) ([]byte, bool) {
	position := sort.Search(len(k), func(i int) bool {
		return bytes.Compare(k[i], visited.firstIncluded) >= 0
	})
	if position == len(k) || bytes.Compare(k[position], visited.firstExcluded) >= 0 {
		return nil, false
	}

	return k[position], true
}
