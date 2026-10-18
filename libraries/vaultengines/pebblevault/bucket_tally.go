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

func bucketTalliesOf(reader pebble.Reader) (map[vault.Name]bucketTally, error) {
	tallies := map[vault.Name]bucketTally{}
	err := amountsWithin(recordCountsRegion, reader).Scan(
		vault.EveryKey(),
		func(tallyKey []byte, records int) (bool, error) {
			tally := tallies[vault.Name(tallyKey)]
			tally.records = records
			tallies[vault.Name(tallyKey)] = tally

			return true, nil
		},
	)
	if err != nil {
		return nil, err
	}
	err = amountsWithin(heldBytesRegion, reader).Scan(
		vault.EveryKey(),
		func(tallyKey []byte, heldBytes int) (bool, error) {
			tally := tallies[vault.Name(tallyKey)]
			tally.heldBytes = int64(heldBytes)
			tallies[vault.Name(tallyKey)] = tally

			return true, nil
		},
	)

	return tallies, err
}

func amountsWithin(region keyspaceRegion, reader pebble.Reader) pebbleAmounts {
	return pebbleAmounts{
		entries: storedEntries{region: region, reader: reader},
		changes: &amountChanges{},
		scans:   &scanTally{},
	}
}
