package expiringrecords_test

import (
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/expiringrecords"
)

var testEpoch = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

type mapRecords map[string][]byte

func (m mapRecords) Get(key []byte) ([]byte, error) { return m[string(key)], nil }

func (m mapRecords) Put(key, record []byte) ([]byte, error) {
	replaced := m[string(key)]
	m[string(key)] = record

	return replaced, nil
}

func (m mapRecords) Delete(key []byte) ([]byte, error) {
	deleted := m[string(key)]
	delete(m, string(key))

	return deleted, nil
}

func (m mapRecords) PutAbsent(key, record []byte) error {
	m[string(key)] = record

	return nil
}

func (m mapRecords) DeleteHeld(key, _ []byte) error {
	delete(m, string(key))

	return nil
}

func (m mapRecords) Len() (int, error) { return len(m), nil }

func (m mapRecords) Scan(keys vault.KeyRange, fn func(key, value []byte) (bool, error)) error {
	firstIncluded, _ := keys.Bounds()
	ordered := make([]string, 0, len(m))
	for key := range m {
		if key >= string(firstIncluded) {
			ordered = append(ordered, key)
		}
	}
	slices.Sort(ordered)
	for _, key := range ordered {
		if keep, err := fn([]byte(key), m[key]); err != nil || !keep {
			return err
		}
	}

	return nil
}

func TestAnExpiryTimeOutsideTheStoredRangeIsRefused(t *testing.T) {
	records := expiringrecords.OverRecords(mapRecords{})

	for _, expiresAt := range []time.Time{
		time.Unix(-1, 0),
		time.Unix(1<<32, 0),
	} {
		if _, err := records.Put([]byte("a"), []byte("record"), expiresAt); err == nil {
			t.Errorf("Put expiring at %s succeeded, want error", expiresAt)
		}
	}
}

func TestAnExpiryTimeRoundsUpToAWholeSecond(t *testing.T) {
	records := expiringrecords.OverRecords(mapRecords{})
	if _, err := records.Put(
		[]byte("a"),
		[]byte("record"),
		testEpoch.Add(time.Millisecond),
	); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if record, err := records.Get([]byte("a"), testEpoch.Add(time.Second-1)); err != nil ||
		record == nil {
		t.Fatalf("Get before the next whole second = %q, %v, want the record", record, err)
	}
	if record, err := records.Get([]byte("a"), testEpoch.Add(time.Second)); err != nil ||
		record != nil {
		t.Fatalf("Get at the next whole second = %q, %v, want absent", record, err)
	}
}

func TestAStoredRecordShorterThanItsExpiryTimeIsRefused(t *testing.T) {
	records := expiringrecords.OverRecords(mapRecords{"a": {1, 2}})

	if _, err := records.Get([]byte("a"), testEpoch); err == nil {
		t.Fatal("Get succeeded, want error")
	}
}
