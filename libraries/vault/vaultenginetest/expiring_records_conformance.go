package vaultenginetest

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const expiringRecordsBucket vault.Name = "expiring_records"

var conformanceEpoch = time.Date(2026, time.October, 22, 12, 0, 0, 0, time.UTC)

func runExpiringRecordsConformance(
	t *testing.T,
	open func(quotaBytes int64) (vault.Engine, error),
) {
	t.Helper()

	t.Run(
		"ExpiringRecordReadsAsAbsentFromItsExpiryTime",
		func(t *testing.T) { expiringRecordReadsAsAbsentFromItsExpiryTime(t, open) },
	)
	t.Run(
		"ExpiringPutReturnsTheStoredRecordItReplacedEvenExpired",
		func(t *testing.T) { expiringPutReturnsTheStoredRecordItReplacedEvenExpired(t, open) },
	)
	t.Run(
		"ExpiringScanVisitsStoredRecordsEvenExpired",
		func(t *testing.T) { expiringScanVisitsStoredRecordsEvenExpired(t, open) },
	)
	t.Run(
		"ExpiredKeysAcrossResumesListEveryExpiredKey",
		func(t *testing.T) { expiredKeysAcrossResumesListEveryExpiredKey(t, open) },
	)
	t.Run(
		"ForgetExpiredForgetsOnlyRecordsStillExpired",
		func(t *testing.T) { forgetExpiredForgetsOnlyRecordsStillExpired(t, open) },
	)
}

func openExpiringRecords(t *testing.T, open func(int64) (vault.Engine, error)) vault.Engine {
	t.Helper()

	engine := openEngine(t, open, 0)
	if err := engine.ProvisionExpiringRecordsBucket(expiringRecordsBucket); err != nil {
		t.Fatalf("ProvisionExpiringRecordsBucket: %v", err)
	}

	return engine
}

func changeExpiringRecords(
	t *testing.T,
	engine vault.Engine,
	change func(vault.EngineExpiringRecords) error,
) {
	t.Helper()

	if err := engine.Update(context.Background(), func(etx vault.EngineTxn) error {
		return change(etx.ExpiringRecords(expiringRecordsBucket))
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func readExpiringRecords(
	t *testing.T,
	engine vault.Engine,
	read func(vault.EngineExpiringRecords) error,
) {
	t.Helper()

	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		return read(etx.ExpiringRecords(expiringRecordsBucket))
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func putExpiring(t *testing.T, engine vault.Engine, expiresAt time.Time, keys ...string) {
	t.Helper()

	changeExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
		for _, key := range keys {
			if _, err := records.Put([]byte(key), []byte("record "+key), expiresAt); err != nil {
				return err
			}
		}

		return nil
	})
}

func expiringRecordAt(
	t *testing.T,
	engine vault.Engine,
	key string,
	at time.Time,
) []byte {
	t.Helper()

	var record []byte
	readExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
		var err error
		record, err = records.Get([]byte(key), at)
		record = bytes.Clone(record)

		return err
	})

	return record
}

func expiringRecordReadsAsAbsentFromItsExpiryTime(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	engine := openExpiringRecords(t, open)
	expiresAt := conformanceEpoch.Add(time.Hour)
	putExpiring(t, engine, expiresAt, "a")

	if record := expiringRecordAt(t, engine, "a", expiresAt.Add(-time.Second)); string(
		record,
	) != "record a" {
		t.Fatalf("Get before its expiry time = %q, want %q", record, "record a")
	}
	for _, at := range []time.Time{expiresAt, expiresAt.Add(time.Second)} {
		if record := expiringRecordAt(t, engine, "a", at); record != nil {
			t.Fatalf("Get at %s = %q, want absent", at, record)
		}
	}
}

func expiringPutReturnsTheStoredRecordItReplacedEvenExpired(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	engine := openExpiringRecords(t, open)
	putExpiring(t, engine, conformanceEpoch.Add(-time.Hour), "a")

	var replaced []byte
	changeExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
		var err error
		replaced, err = records.Put([]byte("a"), []byte("renewed"), conformanceEpoch.Add(time.Hour))
		replaced = bytes.Clone(replaced)

		return err
	})

	if string(replaced) != "record a" {
		t.Fatalf("Put replaced %q, want %q", replaced, "record a")
	}
}

func expiringScanVisitsStoredRecordsEvenExpired(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	engine := openExpiringRecords(t, open)
	putExpiring(t, engine, conformanceEpoch.Add(-time.Hour), "a")
	putExpiring(t, engine, conformanceEpoch.Add(time.Hour), "b")

	var visited []string
	readExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
		return records.Scan(vault.EveryKey(), func(key, record []byte) (bool, error) {
			visited = append(visited, string(key)+"="+string(record))

			return true, nil
		})
	})

	if want := []string{"a=record a", "b=record b"}; !slices.Equal(visited, want) {
		t.Fatalf("Scan visited %v, want %v", visited, want)
	}
}

func expiredKeysAcrossResumesListEveryExpiredKey(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	engine := openExpiringRecords(t, open)
	putExpiring(t, engine, conformanceEpoch, "a", "c", "e", "f")
	putExpiring(t, engine, conformanceEpoch.Add(time.Second), "b", "d", "g")

	var listed []string
	var resume []byte
	for range 10 {
		var expiredKeys [][]byte
		readExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
			var err error
			expiredKeys, resume, err = records.ExpiredKeys(resume, conformanceEpoch, 2)

			return err
		})
		if len(expiredKeys) > 2 {
			t.Fatalf("ExpiredKeys listed %d keys, want at most the limit 2", len(expiredKeys))
		}
		for _, key := range expiredKeys {
			listed = append(listed, string(key))
		}
		if resume == nil {
			break
		}
	}

	if want := []string{"a", "c", "e", "f"}; !slices.Equal(listed, want) {
		t.Fatalf("expired keys = %v, want %v", listed, want)
	}
}

func forgetExpiredForgetsOnlyRecordsStillExpired(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	engine := openExpiringRecords(t, open)
	putExpiring(t, engine, conformanceEpoch, "a", "b")
	putExpiring(t, engine, conformanceEpoch.Add(time.Hour), "b")

	var forgotten []string
	changeExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
		forgotten = nil

		return records.ForgetExpired(
			[][]byte{[]byte("a"), []byte("b"), []byte("missing")},
			conformanceEpoch,
			func(key, forgottenRecord []byte) error {
				forgotten = append(forgotten, string(key)+"="+string(forgottenRecord))

				return nil
			},
		)
	})

	if want := []string{"a=record a"}; !slices.Equal(forgotten, want) {
		t.Fatalf("forgotten = %v, want %v", forgotten, want)
	}
	readExpiringRecords(t, engine, func(records vault.EngineExpiringRecords) error {
		return records.Scan(vault.EveryKey(), func(key, _ []byte) (bool, error) {
			if string(key) != "b" {
				t.Errorf("key %q is still stored, want only b", key)
			}

			return true, nil
		})
	})
}
