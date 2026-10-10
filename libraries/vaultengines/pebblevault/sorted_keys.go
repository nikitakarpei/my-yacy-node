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
	var writtenKeys sortedKeys

	entries := changes.Reader()
	for {
		_, key, _, found, err := entries.Next()
		if err != nil {
			return nil, fmt.Errorf("read staged changes: %w", err)
		}
		if !found {
			break
		}
		writtenKeys = append(writtenKeys, bytes.Clone(key))
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
