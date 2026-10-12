package pebblevault

import (
	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleTxn struct {
	reader         pebble.Reader
	changes        *pebble.Batch
	loweredAmounts *amountKeys
	footprint      *readFootprint
}

func (t pebbleTxn) Writable() bool { return t.changes != nil }

func (t pebbleTxn) Records(name vault.Name) vault.EngineRecords {
	return pebbleRecords{
		entries: t.entriesWithin(recordsRegionOf(name)),
		tally:   storedBucketTallyFor(name, t.reader, t.footprint, t.changes),
	}
}

func (t pebbleTxn) Amounts(name vault.Name) vault.EngineAmounts {
	return pebbleAmounts{
		entries:        t.entriesWithin(amountsRegionOf(name)),
		loweredAmounts: t.loweredAmounts,
	}
}

func (t pebbleTxn) entriesWithin(region keyspaceRegion) storedEntries {
	return storedEntries{
		region:    region,
		reader:    t.reader,
		changes:   t.changes,
		footprint: t.footprint,
	}
}
