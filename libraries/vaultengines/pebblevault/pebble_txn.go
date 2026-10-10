package pebblevault

import (
	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleTxn struct {
	reader  pebble.Reader
	staged  *pebble.Batch
	lowered *loweredAmounts
}

func (t pebbleTxn) Writable() bool { return t.staged != nil }

func (t pebbleTxn) Records(name vault.Name) vault.EngineRecords {
	return pebbleRecords{
		entries: storedEntries{region: recordsRegionOf(name), reader: t.reader},
		tally:   storedBucketTallyFor(name, t.reader, t.staged),
		staged:  t.staged,
	}
}

func (t pebbleTxn) Amounts(name vault.Name) vault.EngineAmounts {
	return pebbleAmounts{
		entries: storedEntries{region: amountsRegionOf(name), reader: t.reader},
		staged:  t.staged,
		lowered: t.lowered,
	}
}
