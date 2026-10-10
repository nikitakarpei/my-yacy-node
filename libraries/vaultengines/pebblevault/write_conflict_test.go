package pebblevault_test

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
)

const (
	conflictBucket  vault.Name = "words"
	conflictAmounts vault.Name = "tallies"
)

type conflictPlace struct {
	bucket vault.Name
	kind   pebblevault.KeyKind
}

type recordedConflicts struct {
	places          []conflictPlace
	exclusiveWrites int
}

func (r *recordedConflicts) ObserveWriteConflicted(
	bucket vault.Name,
	kind pebblevault.KeyKind,
) {
	r.places = append(r.places, conflictPlace{bucket: bucket, kind: kind})
}

func (r *recordedConflicts) ObserveExclusiveWrite() { r.exclusiveWrites++ }

type writeScenario struct {
	storedKeys          []string
	firstCommittedWrite func(vault.EngineTxn) error
	waitingWrite        func(vault.EngineTxn) error
}

func conflictsOf(t *testing.T, scenario writeScenario) []conflictPlace {
	t.Helper()

	conflicts := &recordedConflicts{}
	synctest.Test(t, func(t *testing.T) {
		engine := openObservedEngine(t, conflicts)
		update(t, engine, puttingKeys(scenario.storedKeys...))

		began := make(chan struct{})
		committed := make(chan struct{})
		var writes sync.WaitGroup
		writes.Go(func() {
			update(t, engine, onFirstRun(func() {
				close(began)
				awaitSignal(t, committed)
			}, scenario.waitingWrite))
		})
		awaitSignal(t, began)
		update(t, engine, scenario.firstCommittedWrite)
		close(committed)
		writes.Wait()
	})

	return conflicts.places
}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Hour):
		t.Error("signal never came: a write waited for another that could not run")
	}
}

func openObservedEngine(t *testing.T, conflicts *recordedConflicts) *pebblevault.Engine {
	t.Helper()

	engine, err := pebblevault.OpenEngine(
		filepath.Join(t.TempDir(), "node"),
		0,
		testLimits,
		nil,
		conflicts,
	)
	if err != nil {
		t.Fatalf("OpenEngine: %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	return engine
}

func onFirstRun(
	firstRunOnly func(),
	fn func(vault.EngineTxn) error,
) func(vault.EngineTxn) error {
	var runs int

	return func(tx vault.EngineTxn) error {
		runs++
		if runs == 1 {
			firstRunOnly()
		}

		return fn(tx)
	}
}

func update(t *testing.T, engine *pebblevault.Engine, fn func(vault.EngineTxn) error) {
	t.Helper()

	if err := engine.Update(context.Background(), fn); err != nil {
		t.Errorf("Update: %v", err)
	}
}

func puttingKeys(keys ...string) func(vault.EngineTxn) error {
	return func(tx vault.EngineTxn) error {
		for _, key := range keys {
			if _, err := tx.Records(conflictBucket).Put([]byte(key), []byte("value")); err != nil {
				return err
			}
		}

		return nil
	}
}

func reading(key string) func(vault.EngineTxn) error {
	return func(tx vault.EngineTxn) error {
		_, err := tx.Records(conflictBucket).Get([]byte(key))

		return err
	}
}

func scanningUpTo(visits int) func(vault.EngineTxn) error {
	return func(tx vault.EngineTxn) error {
		var visited int

		return tx.Records(conflictBucket).Scan(vault.EveryKey(), func(_, _ []byte) (bool, error) {
			visited++

			return visited < visits, nil
		})
	}
}

func raising(key string) func(vault.EngineTxn) error {
	return func(tx vault.EngineTxn) error {
		return tx.Amounts(conflictAmounts).Raise([]byte(key), 1)
	}
}

func assertConflicts(t *testing.T, got []conflictPlace, want ...conflictPlace) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("conflicts = %v, want %v", got, want)
	}
}

func TestWriteThatReadsAKeyCommittedWhileItWaitedConflicts(t *testing.T) {
	got := conflictsOf(
		t,
		writeScenario{firstCommittedWrite: puttingKeys("alpha"), waitingWrite: reading("alpha")},
	)

	assertConflicts(
		t,
		got,
		conflictPlace{bucket: conflictBucket, kind: pebblevault.KindRecords},
	)
}

func TestWriteThatReadsAKeyDeletedWhileItWaitedConflicts(t *testing.T) {
	got := conflictsOf(t, writeScenario{
		storedKeys: []string{"alpha"},
		firstCommittedWrite: func(tx vault.EngineTxn) error {
			_, err := tx.Records(conflictBucket).Delete([]byte("alpha"))

			return err
		},
		waitingWrite: reading("alpha"),
	})

	assertConflicts(
		t,
		got,
		conflictPlace{bucket: conflictBucket, kind: pebblevault.KindRecords},
	)
}

func TestWriteThatReadsOtherKeysDoesNotConflict(t *testing.T) {
	got := conflictsOf(
		t,
		writeScenario{firstCommittedWrite: puttingKeys("alpha"), waitingWrite: reading("beta")},
	)

	assertConflicts(t, got)
}

func TestWritesThatPutDifferentKeysOfOneBucketDoNotConflict(t *testing.T) {
	got := conflictsOf(
		t,
		writeScenario{firstCommittedWrite: puttingKeys("alpha"), waitingWrite: puttingKeys("beta")},
	)

	assertConflicts(t, got)
}

func TestScanThatStoppedBeforeACommittedKeyDoesNotConflict(t *testing.T) {
	got := conflictsOf(t, writeScenario{
		storedKeys:          []string{"a", "c"},
		firstCommittedWrite: puttingKeys("b"),
		waitingWrite:        scanningUpTo(1),
	})

	assertConflicts(t, got)
}

func TestScanThatPassedACommittedKeyConflicts(t *testing.T) {
	got := conflictsOf(t, writeScenario{
		storedKeys:          []string{"a", "c"},
		firstCommittedWrite: puttingKeys("b"),
		waitingWrite:        scanningUpTo(3),
	})

	assertConflicts(
		t,
		got,
		conflictPlace{bucket: conflictBucket, kind: pebblevault.KindRecords},
	)
}

func TestBucketLengthAfterAnotherWriteGrewTheBucketConflicts(t *testing.T) {
	got := conflictsOf(t, writeScenario{
		firstCommittedWrite: puttingKeys("alpha"),
		waitingWrite: func(tx vault.EngineTxn) error {
			_, err := tx.Records(conflictBucket).Len()

			return err
		},
	})

	assertConflicts(
		t,
		got,
		conflictPlace{bucket: conflictBucket, kind: pebblevault.KindTallies},
	)
}

func TestRaisesOfOneAmountDoNotConflict(t *testing.T) {
	got := conflictsOf(
		t,
		writeScenario{firstCommittedWrite: raising("the"), waitingWrite: raising("the")},
	)

	assertConflicts(t, got)
}

func TestAmountReadAfterAnotherWriteRaisedItConflicts(t *testing.T) {
	got := conflictsOf(t, writeScenario{
		firstCommittedWrite: raising("the"),
		waitingWrite: func(tx vault.EngineTxn) error {
			_, err := tx.Amounts(conflictAmounts).Get([]byte("the"))

			return err
		},
	})

	assertConflicts(
		t,
		got,
		conflictPlace{bucket: conflictAmounts, kind: pebblevault.KindAmounts},
	)
}

func TestWriteThatReadsAKeyCommittedBeforeItBeganDoesNotConflict(t *testing.T) {
	got := conflictsOf(t, writeScenario{
		storedKeys:          []string{"alpha"},
		firstCommittedWrite: puttingKeys("beta"),
		waitingWrite:        reading("alpha"),
	})

	assertConflicts(t, got)
}

func TestConflictedWriteRunsAgainOnTheCommittedValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := openObservedEngine(t, &recordedConflicts{})
		update(t, engine, putting("count", "1"))

		began := make(chan struct{})
		committed := make(chan struct{})
		var writes sync.WaitGroup
		writes.Go(func() {
			update(t, engine, func(tx vault.EngineTxn) error {
				count, err := tx.Records(conflictBucket).Get([]byte("count"))
				if err != nil {
					return err
				}
				if string(count) == "1" {
					close(began)
					awaitSignal(t, committed)
				}

				return putting("count", string(count)+"+1")(tx)
			})
		})
		awaitSignal(t, began)
		update(t, engine, putting("count", "2"))
		close(committed)
		writes.Wait()

		assertStored(t, engine, "count", "2+1")
	})
}

func TestWritesRunAtTheSameTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := openObservedEngine(t, &recordedConflicts{})

		firstRunning := make(chan struct{})
		var writes sync.WaitGroup
		writes.Go(func() {
			update(t, engine, func(tx vault.EngineTxn) error {
				close(firstRunning)

				return puttingKeys("alpha")(tx)
			})
		})
		update(t, engine, func(tx vault.EngineTxn) error {
			awaitSignal(t, firstRunning)

			return puttingKeys("beta")(tx)
		})
		writes.Wait()
	})
}

func TestWriteThatKeepsConflictingHoldsOtherCommits(t *testing.T) {
	conflicts := &recordedConflicts{}
	synctest.Test(t, func(t *testing.T) {
		engine := openObservedEngine(t, conflicts)

		var others sync.WaitGroup
		update(t, engine, func(tx vault.EngineTxn) error {
			if err := reading("alpha")(tx); err != nil {
				return err
			}
			others.Go(func() { update(t, engine, puttingKeys("alpha")) })
			synctest.Wait()

			return nil
		})
		others.Wait()
	})

	if conflicts.exclusiveWrites != 1 {
		t.Fatalf("exclusive writes = %d, want 1", conflicts.exclusiveWrites)
	}
	if len(conflicts.places) != 3 {
		t.Fatalf("conflicts = %v, want 3 before the write held other commits", conflicts.places)
	}
}

func putting(key, value string) func(vault.EngineTxn) error {
	return func(tx vault.EngineTxn) error {
		_, err := tx.Records(conflictBucket).Put([]byte(key), []byte(value))

		return err
	}
}

func assertStored(t *testing.T, engine *pebblevault.Engine, key, want string) {
	t.Helper()

	if err := engine.View(context.Background(), func(tx vault.EngineTxn) error {
		got, err := tx.Records(conflictBucket).Get([]byte(key))
		if string(got) != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
}
