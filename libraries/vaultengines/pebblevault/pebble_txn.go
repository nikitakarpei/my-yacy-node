package pebblevault

import (
	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleTxn struct {
	reader          pebble.Reader
	changes         *pebble.Batch
	loweredAmounts  *amountKeys
	footprint       *readFootprint
	recordsByBucket map[vault.Name]*pebbleRecords
	amountsByBucket map[vault.Name]*pebbleAmounts
}

func (t *pebbleTxn) Writable() bool { return t.changes != nil }

func (t *pebbleTxn) Records(name vault.Name) vault.EngineRecords {
	if records, opened := t.recordsByBucket[name]; opened {
		return records
	}
	if t.recordsByBucket == nil {
		t.recordsByBucket = map[vault.Name]*pebbleRecords{}
	}
	records := &pebbleRecords{
		entries: t.entriesWithin(recordsRegionOf(name)),
		tally:   storedBucketTallyFor(name, t.reader, t.footprint, t.changes),
	}
	t.recordsByBucket[name] = records

	return records
}

func (t *pebbleTxn) Amounts(name vault.Name) vault.EngineAmounts {
	if amounts, opened := t.amountsByBucket[name]; opened {
		return amounts
	}
	if t.amountsByBucket == nil {
		t.amountsByBucket = map[vault.Name]*pebbleAmounts{}
	}
	amounts := &pebbleAmounts{
		entries:        t.entriesWithin(amountsRegionOf(name)),
		loweredAmounts: t.loweredAmounts,
	}
	t.amountsByBucket[name] = amounts

	return amounts
}

func (t *pebbleTxn) entriesWithin(region keyspaceRegion) storedEntries {
	return storedEntries{
		region:    region,
		reader:    t.reader,
		changes:   t.changes,
		footprint: t.footprint,
	}
}
