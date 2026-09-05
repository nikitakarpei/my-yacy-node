// Package vaultenginetest holds the storage-contract suite every vault Engine driver
// runs against itself. A driver passes its own opener to RunConformance and the
// suite exercises the guarantees a backend must honour: durable round trips,
// bounded scans, transaction atomicity, bucket isolation, and byte accounting.
// Engine-independent behaviour the port enforces lives with the port's own
// tests, not here.
package vaultenginetest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

func RunConformance(t *testing.T, open func(quotaBytes int64) (vault.Engine, error)) {
	t.Helper()

	t.Run("RoundTripAndLength", func(t *testing.T) { roundTripAndLength(t, open) })
	t.Run("MissingKeyReportsAbsent", func(t *testing.T) { missingKeyReportsAbsent(t, open) })
	t.Run(
		"LengthAfterDeleteAndOverwrite",
		func(t *testing.T) { lengthAfterDeleteAndOverwrite(t, open) },
	)
	t.Run("ScanVisitsRangeInOrder", func(t *testing.T) { scanVisitsRangeInOrder(t, open) })
	t.Run("ScanStopsWhenAsked", func(t *testing.T) { scanStopsWhenAsked(t, open) })
	t.Run(
		"BoundedScanVisitsEveryKeyInRange",
		func(t *testing.T) { boundedScanVisitsEveryKeyInRange(t, open) },
	)
	t.Run("BoundedScanStopsWhenAsked", func(t *testing.T) { boundedScanStopsWhenAsked(t, open) })
	t.Run(
		"CrossCollectionAtomicRollback",
		func(t *testing.T) { crossCollectionAtomicRollback(t, open) },
	)
	t.Run(
		"ScanEntriesLastUntilTheClosureReturns",
		func(t *testing.T) { scanEntriesLastUntilTheClosureReturns(t, open) },
	)
	t.Run("DoneContextSkipsTheClosure", func(t *testing.T) { doneContextSkipsTheClosure(t, open) })
	t.Run("BucketOwnershipIsolation", func(t *testing.T) { bucketOwnershipIsolation(t, open) })
	t.Run("AtCapacityTracksQuota", func(t *testing.T) { atCapacityTracksQuota(t, open) })
	t.Run("UsedBytesGrowsWithData", func(t *testing.T) { usedBytesGrowsWithData(t, open) })
	t.Run("UsedBytesFallsAfterDelete", func(t *testing.T) { usedBytesFallsAfterDelete(t, open) })
	t.Run(
		"ConcurrentUpdatesKeepEveryIncrement",
		func(t *testing.T) { concurrentUpdatesKeepEveryIncrement(t, open) },
	)
	t.Run(
		"RepeatedWriteCommitsOneValue",
		func(t *testing.T) { repeatedWriteCommitsOneValue(t, open) },
	)
	t.Run(
		"RepeatedWriteSeesTheSameStoredState",
		func(t *testing.T) { repeatedWriteSeesTheSameStoredState(t, open) },
	)
}

func openVault(
	t *testing.T,
	open func(int64) (vault.Engine, error),
	quotaBytes int64,
) *vault.Vault {
	t.Helper()

	v, err := vault.New(openEngine(t, open, quotaBytes), nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return v
}

func openEngine(
	t *testing.T,
	open func(int64) (vault.Engine, error),
	quotaBytes int64,
) vault.Engine {
	t.Helper()

	opened, err := open(quotaBytes)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return opened
}

func register(t *testing.T, v *vault.Vault, name string) *vault.Collection[string, string] {
	t.Helper()

	collection, err := v.RegisterCollection(
		vault.Name(name),
		stringKeyLayout,
		stringValueCodec{},
	)
	if err != nil {
		t.Fatalf("Register %s: %v", name, err)
	}

	return collection
}

func wrapTest(err error) error {
	return fmt.Errorf("vault op: %w", err)
}

func roundTripAndLength(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	words := register(t, v, "words")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		if err := words.Put(tx, "a", "alpha"); err != nil {
			return wrapTest(err)
		}

		return words.Put(tx, "b", "beta")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		got, ok, err := words.Get(tx, "a")
		if err != nil {
			return wrapTest(err)
		}
		if !ok || got != "alpha" {
			t.Fatalf("Get(a) = %q, %v", got, ok)
		}

		length, err := words.Len(tx)
		if err != nil {
			return wrapTest(err)
		}
		if length != 2 {
			t.Fatalf("Len = %d, want 2", length)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func missingKeyReportsAbsent(t *testing.T, open func(int64) (vault.Engine, error)) {
	v := openVault(t, open, 0)
	words := register(t, v, "words")

	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		_, ok, err := words.Get(tx, "absent")
		if err != nil {
			return wrapTest(err)
		}
		if ok {
			t.Fatal("Get reported a missing key as present")
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func lengthAfterDeleteAndOverwrite(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	words := register(t, v, "words")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		if err := words.Put(tx, "a", "alpha"); err != nil {
			return wrapTest(err)
		}
		if err := words.Put(tx, "a", "again"); err != nil {
			return wrapTest(err)
		}
		deleted, err := words.Delete(tx, "a")
		if err != nil {
			return wrapTest(err)
		}
		if !deleted {
			t.Fatal("Delete reported missing key")
		}
		missing, err := words.Delete(tx, "a")
		if err != nil {
			return wrapTest(err)
		}
		if missing {
			t.Fatal("second Delete reported a deletion")
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		length, err := words.Len(tx)
		if err != nil {
			return wrapTest(err)
		}
		if length != 0 {
			t.Fatalf("Len = %d, want 0", length)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func scanVisitsRangeInOrder(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	words := register(t, v, "words")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		for _, key := range []string{"qa", "pb", "pa"} {
			if err := words.Put(tx, key, key); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var visited []string
	if err := v.View(ctx, func(tx *vault.Txn) error {
		return words.Scan(
			tx,
			stringKeyParts.KeysThroughFirst("pb"),
			func(_ string, value string) (bool, error) {
				visited = append(visited, value)

				return true, nil
			},
		)
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	if len(visited) != 2 || visited[0] != "pa" || visited[1] != "pb" {
		t.Fatalf("scan visited = %v, want [pa pb]", visited)
	}
}

func scanStopsWhenAsked(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	words := register(t, v, "words")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		for _, key := range []string{"a", "b", "c"} {
			if err := words.Put(tx, key, key); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var visited []string
	if err := v.View(ctx, func(tx *vault.Txn) error {
		return words.Scan(tx, vault.EveryKey(), func(_ string, value string) (bool, error) {
			visited = append(visited, value)

			return false, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	if len(visited) != 1 || visited[0] != "a" {
		t.Fatalf("scan visited = %v, want [a]", visited)
	}
}

const boundedScanBucket = vault.Name("bounded")

var boundedScanKeys = []string{"aa", "b1", "cc", "ee"}

func boundedScanVisitsEveryKeyInRange(t *testing.T, open func(int64) (vault.Engine, error)) {
	for _, scenario := range []struct {
		name string
		keys vault.KeyRange
		want []string
	}{
		{"UnboundedOnBothSides", vault.EveryKey(), boundedScanKeys},
		{
			"IncludedLowerBoundOnly",
			stringKeyParts.KeysFromFirst("cc"),
			[]string{"cc", "ee"},
		},
		{
			"ExcludedUpperBoundOnly",
			stringKeyParts.KeysBeforeFirst("cc"),
			[]string{"aa", "b1"},
		},
		{"BothBounds", stringKeyParts.KeysWithFirst("b1"), []string{"b1"}},
		{"BoundsBetweenStoredKeys", stringKeyParts.KeysWithFirst("d"), nil},
		{"UpperBoundBelowEveryKey", stringKeyParts.KeysBeforeFirst(""), nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			visited := scannedKeys(t, open, scenario.keys)
			if !slices.Equal(visited, scenario.want) {
				t.Fatalf("bounded scan visited = %v, want %v", visited, scenario.want)
			}
		})
	}
}

func scannedKeys(
	t *testing.T,
	open func(int64) (vault.Engine, error),
	keys vault.KeyRange,
) []string {
	t.Helper()

	engine := openEngine(t, open, 0)
	storeBoundedScanKeys(t, engine)

	var visited []string
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		return etx.Bucket(boundedScanBucket).Scan(keys, func(key, _ []byte) (bool, error) {
			decoded, err := stringKeyLayout.Decode(key)
			if err != nil {
				return false, err
			}
			visited = append(visited, decoded)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return visited
}

func storeBoundedScanKeys(t *testing.T, engine vault.Engine) {
	t.Helper()

	if err := engine.Provision(boundedScanBucket); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if err := engine.Update(context.Background(), func(etx vault.EngineTxn) error {
		bucket := etx.Bucket(boundedScanBucket)
		for _, key := range boundedScanKeys {
			if err := bucket.Put(stringKeyParts.Key(key).Bytes(), []byte(key)); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func boundedScanStopsWhenAsked(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openEngine(t, open, 0)
	storeBoundedScanKeys(t, engine)

	var visited []string
	if err := engine.View(context.Background(), func(etx vault.EngineTxn) error {
		return etx.Bucket(boundedScanBucket).Scan(
			vault.EveryKey(),
			func(key, _ []byte) (bool, error) {
				decoded, err := stringKeyLayout.Decode(key)
				if err != nil {
					return false, err
				}
				visited = append(visited, decoded)

				return false, nil
			},
		)
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	if want := boundedScanKeys[:1]; !slices.Equal(visited, want) {
		t.Fatalf("bounded scan visited = %v, want %v", visited, want)
	}
}

func crossCollectionAtomicRollback(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	left := register(t, v, "left")
	right := register(t, v, "right")

	sentinel := errors.New("boom")
	err := v.Update(ctx, func(tx *vault.Txn) error {
		if err := left.Put(tx, "a", "alpha"); err != nil {
			return wrapTest(err)
		}
		if err := right.Put(tx, "b", "beta"); err != nil {
			return wrapTest(err)
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Update error = %v, want sentinel", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		leftLen, err := left.Len(tx)
		if err != nil {
			return wrapTest(err)
		}
		rightLen, err := right.Len(tx)
		if err != nil {
			return wrapTest(err)
		}
		if leftLen != 0 || rightLen != 0 {
			t.Fatalf("lengths after rollback = %d, %d, want 0, 0", leftLen, rightLen)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

const scannedBucket = vault.Name("words")

func scanEntriesLastUntilTheClosureReturns(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	engine := openEngine(t, open, 4096)
	storeScannedEntries(t, engine)

	if err := engine.View(ctx, func(tx vault.EngineTxn) error {
		return tx.Bucket(scannedBucket).Scan(
			vault.EveryKey(),
			func(key, value []byte) (bool, error) {
				entered, valueEntered := string(key), string(value)
				if _, err := tx.Bucket(scannedBucket).Get([]byte("c")); err != nil {
					return false, wrapTest(err)
				}
				if string(key) != entered || string(value) != valueEntered {
					return false, fmt.Errorf(
						"entry %q=%q became %q=%q before the closure returned",
						entered, valueEntered, key, value,
					)
				}

				return true, nil
			},
		)
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func storeScannedEntries(t *testing.T, engine vault.Engine) {
	t.Helper()

	if err := engine.Provision(scannedBucket); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if err := engine.Update(context.Background(), func(tx vault.EngineTxn) error {
		for _, key := range []string{"a", "b", "c"} {
			if err := tx.Bucket(scannedBucket).Put([]byte(key), []byte(key+"-value")); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func doneContextSkipsTheClosure(t *testing.T, open func(int64) (vault.Engine, error)) {
	engine := openEngine(t, open, 4096)
	if err := engine.Provision(scannedBucket); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	verifyClosureSkipped(t, "Update", func(fn func(vault.EngineTxn) error) error {
		return engine.Update(ctx, fn)
	})
	verifyClosureSkipped(t, "View", func(fn func(vault.EngineTxn) error) error {
		return engine.View(ctx, fn)
	})
}

func verifyClosureSkipped(
	t *testing.T,
	operation string,
	run func(func(vault.EngineTxn) error) error,
) {
	t.Helper()

	called := false
	if err := run(func(vault.EngineTxn) error {
		called = true

		return nil
	}); err == nil {
		t.Fatalf("%s on a done context succeeded, want an error", operation)
	}
	if called {
		t.Fatalf("%s called the closure on a done context", operation)
	}
}

func bucketOwnershipIsolation(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	left := register(t, v, "left")
	right := register(t, v, "right")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return left.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		_, ok, err := right.Get(tx, "a")
		if err != nil {
			return wrapTest(err)
		}
		if ok {
			t.Fatal("right collection saw left collection's key")
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func atCapacityTracksQuota(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 1)
	words := register(t, v, "words")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	atCapacity, err := v.AtCapacity(ctx)
	if err != nil {
		t.Fatalf("AtCapacity: %v", err)
	}
	if !atCapacity {
		t.Fatal("AtCapacity = false, want true over quota")
	}

	used, err := v.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if used < v.QuotaBytes() {
		t.Fatalf("UsedBytes = %d, want >= quota %d", used, v.QuotaBytes())
	}
}

func usedBytesGrowsWithData(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 4096)
	words := register(t, v, "words")

	before, err := v.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "a", "alpha")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	after, err := v.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if after < before {
		t.Fatalf("UsedBytes shrank: before=%d after=%d", before, after)
	}
	if v.QuotaBytes() != 4096 {
		t.Fatalf("QuotaBytes = %d, want 4096", v.QuotaBytes())
	}
}

const (
	deletedRowCount      = 512
	deletedRowValueBytes = 512
)

func usedBytesFallsAfterDelete(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openVault(t, open, 16<<20)
	words := register(t, v, "words")

	storeDeletedRows(t, v, words)
	filled, err := v.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		for row := range deletedRowCount {
			if _, err := words.Delete(tx, strconv.Itoa(row)); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	emptied, err := v.UsedBytes(ctx)
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}
	if emptied >= filled {
		t.Fatalf("UsedBytes did not fall: filled=%d emptied=%d", filled, emptied)
	}
}

func storeDeletedRows(t *testing.T, v *vault.Vault, words *vault.Collection[string, string]) {
	t.Helper()

	value := strings.Repeat("x", deletedRowValueBytes)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		for row := range deletedRowCount {
			if err := words.Put(tx, strconv.Itoa(row), value); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

const (
	concurrentWriters   = 8
	incrementsPerWriter = 64
	counterKey          = "counter"
)

func concurrentUpdatesKeepEveryIncrement(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	ctx := context.Background()
	v := openVault(t, open, 0)
	counters := register(t, v, "counters")

	for err := range incrementedConcurrently(ctx, v, counters) {
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	wanted := strconv.Itoa(concurrentWriters * incrementsPerWriter)
	if err := v.View(ctx, func(tx *vault.Txn) error {
		counted, _, err := counters.Get(tx, counterKey)
		if err != nil {
			return wrapTest(err)
		}
		if counted != wanted {
			t.Fatalf("counter = %q, want %s", counted, wanted)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func incrementedConcurrently(
	ctx context.Context,
	v *vault.Vault,
	counters *vault.Collection[string, string],
) <-chan error {
	failures := make(chan error, concurrentWriters)
	release := make(chan struct{})

	var writing sync.WaitGroup
	writing.Add(concurrentWriters)
	for range concurrentWriters {
		go func() {
			defer writing.Done()
			<-release
			for range incrementsPerWriter {
				if err := v.Update(ctx, func(tx *vault.Txn) error {
					return incrementCounter(tx, counters)
				}); err != nil {
					failures <- err

					return
				}
			}
		}()
	}
	close(release)
	writing.Wait()
	close(failures)

	return failures
}

func incrementCounter(tx *vault.Txn, counters *vault.Collection[string, string]) error {
	counted, found, err := counters.Get(tx, counterKey)
	if err != nil {
		return wrapTest(err)
	}

	current := 0
	if found {
		current, err = strconv.Atoi(counted)
		if err != nil {
			return fmt.Errorf("counter value %q: %w", counted, err)
		}
	}

	if err := counters.Put(tx, counterKey, strconv.Itoa(current+1)); err != nil {
		return wrapTest(err)
	}

	return nil
}

func openRepeatingVault(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) *vault.Vault {
	t.Helper()

	v, err := vault.New(EngineRepeatingWrites(openEngine(t, open, 0)), nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return v
}

func repeatedWriteCommitsOneValue(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openRepeatingVault(t, open)
	words := register(t, v, "words")

	if err := v.Update(ctx, func(tx *vault.Txn) error {
		return words.Put(tx, "alpha", "first")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := v.View(ctx, func(tx *vault.Txn) error {
		length, err := words.Len(tx)
		if err != nil {
			return wrapTest(err)
		}
		if length != 1 {
			t.Fatalf("length = %d, want 1", length)
		}

		return nil
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func repeatedWriteSeesTheSameStoredState(t *testing.T, open func(int64) (vault.Engine, error)) {
	ctx := context.Background()
	v := openRepeatingVault(t, open)
	words := register(t, v, "words")

	var found []bool
	if err := v.Update(ctx, func(tx *vault.Txn) error {
		_, stored, err := words.Get(tx, "alpha")
		if err != nil {
			return wrapTest(err)
		}
		found = append(found, stored)

		return words.Put(tx, "alpha", "first")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if len(found) < 2 {
		t.Fatalf("closure runs = %d, want the repeated write to run it more than once", len(found))
	}
	for run, stored := range found {
		if stored {
			t.Fatalf("run %d found a value, want each run to start from the stored state", run)
		}
	}
}
