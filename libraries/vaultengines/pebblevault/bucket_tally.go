package pebblevault

import (
	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	recordCountsPrefix byte = 1
	heldBytesPrefix    byte = 2
)

var (
	recordCountsRegion = keyspaceRegion{prefix: []byte{tallyRegionPrefix, recordCountsPrefix}}
	heldBytesRegion    = keyspaceRegion{prefix: []byte{tallyRegionPrefix, heldBytesPrefix}}
)

type bucketTally struct {
	records   int
	heldBytes int64
}

type storedBucketTally struct {
	recordCounts pebbleAmounts
	heldBytes    pebbleAmounts
	tallyKey     []byte
}

func (t storedBucketTally) value() (bucketTally, error) {
	records, err := t.recordCounts.Get(t.tallyKey)
	if err != nil {
		return bucketTally{}, err
	}
	heldBytes, err := t.heldBytes.Get(t.tallyKey)
	if err != nil {
		return bucketTally{}, err
	}

	return bucketTally{records: records, heldBytes: int64(heldBytes)}, nil
}

func (t storedBucketTally) adjustBy(recordsDelta int, heldBytesDelta int64) {
	t.recordCounts.change(t.tallyKey, int64(recordsDelta))
	t.heldBytes.change(t.tallyKey, heldBytesDelta)
}

func tallyBucketFrom(tallyKey []byte) vault.Name {
	return vault.Name(tallyKey[len(recordCountsRegion.prefix):])
}

func heldBytesOf(reader pebble.Reader) (int64, error) {
	var heldBytes int64

	heldBytesAmounts := pebbleAmounts{
		entries: storedEntries{region: heldBytesRegion, reader: reader},
		changes: &amountChanges{},
		scans:   &scanTally{},
	}
	err := heldBytesAmounts.Scan(
		vault.EveryKey(),
		func(_ []byte, bytes int) (bool, error) {
			heldBytes += int64(bytes)

			return true, nil
		},
	)

	return heldBytes, err
}
