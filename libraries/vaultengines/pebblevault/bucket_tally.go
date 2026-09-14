package pebblevault

import (
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var tallyRegion = keyspaceRegion{prefix: []byte{tallyRegionPrefix}}

var errBadBucketTally = errors.New("bad bucket tally")

type bucketTally struct {
	entries int
	bytes   int64
}

func (t bucketTally) adjustedBy(entriesDelta int, bytesDelta int64) bucketTally {
	return bucketTally{
		entries: t.entries + entriesDelta,
		bytes:   t.bytes + bytesDelta,
	}
}

type storedBucketTally struct {
	tallies storedEntries
	bucket  vault.Name
	staged  *pebble.Batch
}

func storedBucketTallyFor(
	bucket vault.Name,
	reader pebble.Reader,
	staged *pebble.Batch,
) storedBucketTally {
	return storedBucketTally{
		tallies: bucketTalliesIn(reader),
		bucket:  bucket,
		staged:  staged,
	}
}

func bucketTalliesIn(reader pebble.Reader) storedEntries {
	return storedEntries{region: tallyRegion, reader: reader}
}

func (t storedBucketTally) value() (bucketTally, error) {
	raw, err := t.tallies.valueAt([]byte(t.bucket))
	if err != nil {
		return bucketTally{}, err
	}

	return decodedBucketTally(raw)
}

func decodedBucketTally(raw []byte) (bucketTally, error) {
	stored := storedfields.ReaderOf(raw, errBadBucketTally)
	tally := bucketTally{
		entries: stored.Count("bucket entries"),
		bytes:   stored.Varint("bucket bytes"),
	}
	if err := stored.Err(); err != nil {
		return bucketTally{}, err
	}

	return tally, nil
}

func (t storedBucketTally) adjustBy(entriesDelta int, bytesDelta int64) error {
	current, err := t.value()
	if err != nil {
		return err
	}

	if err := t.staged.Set(
		t.tallies.region.absoluteKeyFrom([]byte(t.bucket)),
		encodedBucketTally(current.adjustedBy(entriesDelta, bytesDelta)),
		pebble.NoSync,
	); err != nil {
		return fmt.Errorf("store bucket tally: %w", err)
	}

	return nil
}

func encodedBucketTally(tally bucketTally) []byte {
	var stored storedfields.Writer
	stored.Count(max(tally.entries, 0))
	stored.Varint(max(tally.bytes, 0))

	return stored.Record()
}

func heldBytesOf(tallies storedEntries) (int64, error) {
	var held int64

	if err := tallies.visit(vault.EveryKey(), func(_, value []byte) (bool, error) {
		tally, err := decodedBucketTally(value)
		if err != nil {
			return false, err
		}
		held += tally.bytes

		return true, nil
	}); err != nil {
		return 0, err
	}

	return held, nil
}
