// Package expiringrecords keeps the expiring records of an engine that has no expiry
// of its own in a plain records bucket. Each stored record starts with its expiry
// time in whole seconds, rounded up. An expired record reads as absent and stays
// stored until vault asks to forget it; finding expired records reads the bucket.
package expiringrecords

import (
	"bytes"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func OverRecords(records vault.EngineRecords) vault.EngineExpiringRecords {
	return expiringOverRecords{records: records}
}

type expiringOverRecords struct {
	records vault.EngineRecords
}

func (r expiringOverRecords) Get(key []byte, at time.Time) ([]byte, error) {
	stored, err := r.records.Get(key)
	if err != nil || stored == nil {
		return nil, err
	}
	expiresAt, record, err := expiryAndRecordOf(stored)
	if err != nil || expiresAt.hasPassedAt(at) {
		return nil, err
	}

	return record, nil
}

func (r expiringOverRecords) Put(key, record []byte, expiresAt time.Time) ([]byte, error) {
	storedExpiry, err := expiryFrom(expiresAt)
	if err != nil {
		return nil, err
	}
	replaced, err := r.records.Put(key, storedFrom(storedExpiry, record))
	if err != nil || replaced == nil {
		return nil, err
	}
	_, replacedRecord, err := expiryAndRecordOf(replaced)

	return replacedRecord, err
}

func (r expiringOverRecords) Scan(
	keys vault.KeyRange,
	fn func(key, record []byte) (bool, error),
) error {
	return r.records.Scan(keys, func(key, stored []byte) (bool, error) {
		_, record, err := expiryAndRecordOf(stored)
		if err != nil {
			return false, err
		}

		return fn(key, record)
	})
}

func (r expiringOverRecords) Len() (int, error) {
	return r.records.Len()
}

func (r expiringOverRecords) ExpiredKeys(
	resume []byte,
	at time.Time,
	limit int,
) (expiredKeys [][]byte, nextResume []byte, err error) {
	read := 0
	err = r.records.Scan(vault.KeysFrom(resume), func(key, stored []byte) (bool, error) {
		if read == limit {
			nextResume = bytes.Clone(key)

			return false, nil
		}
		read++
		expiresAt, _, err := expiryAndRecordOf(stored)
		if err != nil {
			return false, err
		}
		if expiresAt.hasPassedAt(at) {
			expiredKeys = append(expiredKeys, bytes.Clone(key))
		}

		return true, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return expiredKeys, nextResume, nil
}

func (r expiringOverRecords) ForgetExpired(
	keys [][]byte,
	at time.Time,
	fn func(key, forgottenRecord []byte) error,
) error {
	for _, key := range keys {
		if err := r.forgetIfExpired(key, at, fn); err != nil {
			return err
		}
	}

	return nil
}

func (r expiringOverRecords) forgetIfExpired(
	key []byte,
	at time.Time,
	fn func(key, forgottenRecord []byte) error,
) error {
	stored, err := r.records.Get(key)
	if err != nil || stored == nil {
		return err
	}
	expiresAt, record, err := expiryAndRecordOf(stored)
	if err != nil || !expiresAt.hasPassedAt(at) {
		return err
	}
	if err := r.records.DeleteHeld(key, stored); err != nil {
		return err
	}

	return fn(key, record)
}
