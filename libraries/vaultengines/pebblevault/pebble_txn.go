package pebblevault

import (
	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type pebbleTxn struct {
	reader pebble.Reader
	staged *pebble.Batch
}

func (t pebbleTxn) Writable() bool { return t.staged != nil }

func (t pebbleTxn) Bucket(name vault.Name) vault.EngineBucket {
	return pebbleBucket{
		entries: storedEntries{region: bucketRegionOf(name), reader: t.reader},
		tally:   storedBucketTallyFor(name, t.reader, t.staged),
		staged:  t.staged,
	}
}
