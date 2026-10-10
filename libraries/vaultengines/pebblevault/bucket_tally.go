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

func storedBucketTallyFor(
	bucket vault.Name,
	reader pebble.Reader,
	staged *pebble.Batch,
) storedBucketTally {
	return storedBucketTally{
		recordCounts: tallyAmountsWithin(recordCountsRegion, reader, staged),
		heldBytes:    tallyAmountsWithin(heldBytesRegion, reader, staged),
		tallyKey:     []byte(bucket),
	}
}

func tallyAmountsWithin(
	region keyspaceRegion,
	reader pebble.Reader,
	staged *pebble.Batch,
) pebbleAmounts {
	return pebbleAmounts{
		entries: storedEntries{region: region, reader: reader},
		staged:  staged,
		lowered: &loweredAmounts{},
	}
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

func (t storedBucketTally) adjustBy(recordsDelta int, heldBytesDelta int64) error {
	if recordsDelta != 0 {
		if err := t.recordCounts.stageChange(t.tallyKey, int64(recordsDelta)); err != nil {
			return err
		}
	}
	if heldBytesDelta != 0 {
		return t.heldBytes.stageChange(t.tallyKey, heldBytesDelta)
	}

	return nil
}

func heldBytesOf(reader pebble.Reader) (int64, error) {
	var held int64

	err := tallyAmountsWithin(heldBytesRegion, reader, nil).Scan(
		vault.EveryKey(),
		func(_ []byte, bytes int) (bool, error) {
			held += int64(bytes)

			return true, nil
		},
	)

	return held, err
}
