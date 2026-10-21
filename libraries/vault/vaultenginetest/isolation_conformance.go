package vaultenginetest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	concurrentReaders       = 4
	contendedRounds         = 64
	dueEntriesToLease       = 32
	leasedUntil       int64 = 1 << 40
	contendedKey            = "visit"
)

var errTornRead = errors.New("transaction read a state no commit left")

func runIsolationConformance(t *testing.T, open func(quotaBytes int64) (vault.Engine, error)) {
	t.Helper()

	t.Run(
		"UpdateReadsOneCommittedState",
		func(t *testing.T) { updateReadsOneCommittedState(t, open) },
	)
	t.Run(
		"ViewReadsOneCommittedState",
		func(t *testing.T) { viewReadsOneCommittedState(t, open) },
	)
	t.Run(
		"ConcurrentLeasesTakeEachDueEntryOnce",
		func(t *testing.T) { concurrentLeasesTakeEachDueEntryOnce(t, open) },
	)
	t.Run(
		"LeasesAndReturnsOfOneEntryKeepEveryMove",
		func(t *testing.T) { leasesAndReturnsOfOneEntryKeepEveryMove(t, open) },
	)
	t.Run(
		"LeaseAgainstDeleteLeavesNoEntry",
		func(t *testing.T) { leaseAgainstDeleteLeavesNoEntry(t, open) },
	)
	t.Run(
		"ConcurrentFirstInsertsLeaveOneWinner",
		func(t *testing.T) { concurrentFirstInsertsLeaveOneWinner(t, open) },
	)
}

func updateReadsOneCommittedState(t *testing.T, open func(int64) (vault.Engine, error)) {
	writer, reader := openTwoHandles(t, open)

	if err := pairReadWhileChanged(writer, reader, reader.Update); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func openTwoHandles(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) (first, second *vault.Vault) {
	t.Helper()

	engine := openEngine(t, open, 0)
	first, err := vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	second, err = vault.New(engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return first, second
}

func pairReadWhileChanged(
	writer, reader *vault.Vault,
	read func(context.Context, func(*vault.Txn) error) error,
) error {
	writtenPair, err := writer.RegisterCollection(
		vault.Name("pair"),
		stringKeyLayout,
		stringValueCodec{},
	)
	if err != nil {
		return fmt.Errorf("register pair: %w", err)
	}
	readPair, err := reader.RegisterCollection(
		vault.Name("pair"),
		stringKeyLayout,
		stringValueCodec{},
	)
	if err != nil {
		return fmt.Errorf("register pair: %w", err)
	}

	var work []func() error
	for range concurrentWriters {
		work = append(work, func() error {
			return repeated(func() error {
				return writer.Update(context.Background(), func(tx *vault.Txn) error {
					return raisePair(tx, writtenPair)
				})
			})
		})
	}
	for range concurrentReaders {
		work = append(work, func() error {
			return repeated(func() error {
				return read(context.Background(), func(tx *vault.Txn) error {
					return readOnePairState(tx, readPair)
				})
			})
		})
	}

	return failuresOfConcurrent(work)
}

func repeated(run func() error) error {
	for range incrementsPerWriter {
		if err := run(); err != nil {
			return err
		}
	}

	return nil
}

func raisePair(tx *vault.Txn, pair *vault.Collection[string, string]) error {
	left, _, err := pair.Get(tx, "left")
	if err != nil {
		return wrapTest(err)
	}
	raised := left + "x"
	if _, err := pair.Put(tx, "left", raised); err != nil {
		return wrapTest(err)
	}
	if _, err := pair.Put(tx, "right", raised); err != nil {
		return wrapTest(err)
	}

	return nil
}

func readOnePairState(tx *vault.Txn, pair *vault.Collection[string, string]) error {
	left, _, err := pair.Get(tx, "left")
	if err != nil {
		return wrapTest(err)
	}
	scanned := map[string]string{}
	if err := pair.Scan(tx, vault.EveryKey(), func(key, value string) (bool, error) {
		scanned[key] = value

		return true, nil
	}); err != nil {
		return wrapTest(err)
	}
	if scanned["left"] != left || scanned["right"] != left {
		return fmt.Errorf("%w: left %q, scanned %v", errTornRead, left, scanned)
	}

	return nil
}

func failuresOfConcurrent(work []func() error) error {
	failures := make([]error, len(work))
	release := make(chan struct{})

	var running sync.WaitGroup
	for position, run := range work {
		running.Go(func() {
			<-release
			failures[position] = run()
		})
	}
	close(release)
	running.Wait()

	return errors.Join(failures...)
}

func viewReadsOneCommittedState(t *testing.T, open func(int64) (vault.Engine, error)) {
	writer, reader := openTwoHandles(t, open)

	if err := pairReadWhileChanged(writer, reader, reader.View); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func concurrentLeasesTakeEachDueEntryOnce(t *testing.T, open func(int64) (vault.Engine, error)) {
	first, second := openTwoHandles(t, open)
	schedules := []schedule{openSchedule(t, first), openSchedule(t, second)}
	keys := numberedWords(dueEntriesToLease)
	schedules[0].write(t, func(tx *vault.Txn) error {
		for due, key := range keys {
			if _, err := schedules[0].entries.Put(tx, key, scheduled{due: int64(due)}); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	})

	leasedByWorker := make([][]string, concurrentWriters)
	work := make([]func() error, concurrentWriters)
	for worker := range work {
		work[worker] = func() error {
			for {
				key, leased, err := schedules[worker%len(schedules)].leaseDue()
				if err != nil || !leased {
					return err
				}
				leasedByWorker[worker] = append(leasedByWorker[worker], key)
			}
		}
	}
	if err := failuresOfConcurrent(work); err != nil {
		t.Fatalf("Update: %v", err)
	}

	leased := slices.Concat(leasedByWorker...)
	slices.Sort(leased)
	if !slices.Equal(leased, keys) {
		t.Fatalf("leased = %v, want each of %v once", leased, keys)
	}
}

type schedule struct {
	vault   *vault.Vault
	entries *vault.Collection[string, scheduled]
	byDue   *vault.ValueIndex[dueEntry, dueEntry]
}

type scheduled struct {
	due   int64
	moves int
}

type dueEntry struct {
	due int64
	key string
}

var dueEntries = vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart).KeyLayoutFor(
	func(entry dueEntry) (int64, string) { return entry.due, entry.key },
	func(due int64, key string) dueEntry { return dueEntry{due: due, key: key} },
)

func openSchedule(t *testing.T, v *vault.Vault) schedule {
	t.Helper()

	byDue, err := v.RegisterValueIndex(
		vault.Name("entries_by_due"),
		"1",
		dueEntries,
		func(entry dueEntry) dueEntry { return entry },
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	entries, err := v.RegisterProjectedCollection(
		context.Background(),
		vault.Name("entries"),
		stringKeyLayout,
		scheduledCodec{},
		vault.ProjectedAs(
			func(key string, entry scheduled) dueEntry { return dueEntry{due: entry.due, key: key} },
			byDue,
		),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}

	return schedule{vault: v, entries: entries, byDue: byDue}
}

type scheduledCodec struct{}

func (scheduledCodec) Encode(entry scheduled) ([]byte, error) {
	return fmt.Appendf(nil, "%d/%d", entry.due, entry.moves), nil
}

func (scheduledCodec) Decode(raw []byte) (scheduled, error) {
	var entry scheduled
	if _, err := fmt.Sscanf(string(raw), "%d/%d", &entry.due, &entry.moves); err != nil {
		return scheduled{}, fmt.Errorf("decode scheduled entry %q: %w", raw, err)
	}

	return entry, nil
}

func (s schedule) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := s.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (s schedule) leaseDue() (key string, leased bool, err error) {
	err = s.vault.Update(context.Background(), func(tx *vault.Txn) error {
		key, leased = "", false
		head, found, err := s.headOf(tx)
		if err != nil || !found || head.due >= leasedUntil {
			return err
		}
		key, leased = head.key, true

		return s.move(tx, head.key, leasedUntil)
	})

	return key, leased, err
}

func (s schedule) headOf(tx *vault.Txn) (dueEntry, bool, error) {
	var head dueEntry

	var found bool
	if err := s.byDue.Scan(tx, vault.EveryKey(), func(entry dueEntry) (bool, error) {
		head, found = entry, true

		return false, nil
	}); err != nil {
		return dueEntry{}, false, wrapTest(err)
	}

	return head, found, nil
}

func (s schedule) move(tx *vault.Txn, key string, due int64) error {
	entry, found, err := s.entries.Get(tx, key)
	if err != nil {
		return wrapTest(err)
	}
	if !found {
		return nil
	}
	if _, err := s.entries.Put(tx, key, scheduled{due: due, moves: entry.moves + 1}); err != nil {
		return wrapTest(err)
	}

	return nil
}

func leasesAndReturnsOfOneEntryKeepEveryMove(t *testing.T, open func(int64) (vault.Engine, error)) {
	first, second := openTwoHandles(t, open)
	lessor, returner := openSchedule(t, first), openSchedule(t, second)
	lessor.write(t, func(tx *vault.Txn) error {
		_, err := lessor.entries.Put(tx, contendedKey, scheduled{})

		return err
	})

	work := make([]func() error, 0, concurrentWriters)
	for range concurrentWriters / 2 {
		work = append(
			work,
			func() error { return repeated(lessor.leaseHead) },
			func() error { return repeated(returner.returnContended) },
		)
	}
	if err := failuresOfConcurrent(work); err != nil {
		t.Fatalf("Update: %v", err)
	}

	entry := lessor.entryAt(t, contendedKey)
	if entry.moves != concurrentWriters*incrementsPerWriter {
		t.Fatalf("moves = %d, want %d", entry.moves, concurrentWriters*incrementsPerWriter)
	}
	want := []dueEntry{{due: entry.due, key: contendedKey}}
	if got := lessor.dueEntries(t); !slices.Equal(got, want) {
		t.Fatalf("due entries = %v, want %v", got, want)
	}
}

func (s schedule) leaseHead() error {
	return s.vault.Update(context.Background(), func(tx *vault.Txn) error {
		head, found, err := s.headOf(tx)
		if err != nil || !found {
			return err
		}
		entry, _, err := s.entries.Get(tx, head.key)
		if err != nil {
			return wrapTest(err)
		}

		return s.move(tx, head.key, leasedUntil+int64(entry.moves))
	})
}

func (s schedule) returnContended() error {
	return s.vault.Update(context.Background(), func(tx *vault.Txn) error {
		entry, _, err := s.entries.Get(tx, contendedKey)
		if err != nil {
			return wrapTest(err)
		}

		return s.move(tx, contendedKey, int64(entry.moves))
	})
}

func (s schedule) entryAt(t *testing.T, key string) scheduled {
	t.Helper()

	var entry scheduled
	if err := s.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		entry, _, err = s.entries.Get(tx, key)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return entry
}

func (s schedule) dueEntries(t *testing.T) []dueEntry {
	t.Helper()

	var entries []dueEntry
	if err := s.vault.View(context.Background(), func(tx *vault.Txn) error {
		return s.byDue.Scan(tx, vault.EveryKey(), func(entry dueEntry) (bool, error) {
			entries = append(entries, entry)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return entries
}

func leaseAgainstDeleteLeavesNoEntry(t *testing.T, open func(int64) (vault.Engine, error)) {
	first, second := openTwoHandles(t, open)
	lessor, deleter := openSchedule(t, first), openSchedule(t, second)

	for round := range contendedRounds {
		key := strconv.Itoa(round)
		lessor.write(t, func(tx *vault.Txn) error {
			_, err := lessor.entries.Put(tx, key, scheduled{})

			return err
		})
		if err := failuresOfConcurrent([]func() error{
			func() error {
				_, _, err := lessor.leaseDue()

				return err
			},
			func() error { return deleter.deleteEntry(key) },
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	if got := lessor.dueEntries(t); len(got) != 0 {
		t.Fatalf("due entries after deletes = %v, want none", got)
	}
	if length := lessor.length(t); length != 0 {
		t.Fatalf("entries after deletes = %d, want 0", length)
	}
}

func (s schedule) deleteEntry(key string) error {
	return s.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := s.entries.Delete(tx, key)

		return err
	})
}

func (s schedule) length(t *testing.T) int {
	t.Helper()

	var length int
	if err := s.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		length, err = s.entries.Len(tx)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return length
}

func concurrentFirstInsertsLeaveOneWinner(t *testing.T, open func(int64) (vault.Engine, error)) {
	first, second := openTwoHandles(t, open)
	handles := []*vault.Vault{first, second}
	claims := []*vault.Collection[string, string]{
		register(t, first, "claims"),
		register(t, second, "claims"),
	}

	for round := range contendedRounds {
		key := strconv.Itoa(round)
		inserted := make([]bool, concurrentWriters)
		work := make([]func() error, concurrentWriters)
		for writer := range work {
			work[writer] = func() error {
				return handles[writer%len(handles)].Update(
					context.Background(),
					func(tx *vault.Txn) error {
						var err error
						inserted[writer], err = insertAbsent(
							tx,
							claims[writer%len(claims)],
							key,
							strconv.Itoa(writer),
						)

						return err
					},
				)
			}
		}
		if err := failuresOfConcurrent(work); err != nil {
			t.Fatalf("Update: %v", err)
		}

		winner := slices.Index(inserted, true)
		if winner < 0 || slices.Index(inserted[winner+1:], true) >= 0 {
			t.Fatalf("round %d inserted = %v, want one winner", round, inserted)
		}
		if stored := storedClaim(t, first, claims[0], key); stored != strconv.Itoa(winner) {
			t.Fatalf("round %d stored %q, want winner %d", round, stored, winner)
		}
	}
}

func insertAbsent(
	tx *vault.Txn,
	claims *vault.Collection[string, string],
	key, claimant string,
) (bool, error) {
	_, found, err := claims.Get(tx, key)
	if err != nil || found {
		return false, err
	}
	if _, err := claims.Put(tx, key, claimant); err != nil {
		return false, wrapTest(err)
	}

	return true, nil
}

func storedClaim(
	t *testing.T,
	v *vault.Vault,
	claims *vault.Collection[string, string],
	key string,
) string {
	t.Helper()

	var claimant string
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		claimant, _, err = claims.Get(tx, key)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return claimant
}
