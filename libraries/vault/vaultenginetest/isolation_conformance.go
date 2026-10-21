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
	recordsToMove           = 32
	movedRank         int64 = 1 << 40
	contendedKey            = "contended"
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
		"ConcurrentMovesOfTheFirstIndexEntryMoveEachRecordOnce",
		func(t *testing.T) { concurrentMovesOfTheFirstIndexEntryMoveEachRecordOnce(t, open) },
	)
	t.Run(
		"MovesThroughTheIndexAndByKeyKeepEveryChange",
		func(t *testing.T) { movesThroughTheIndexAndByKeyKeepEveryChange(t, open) },
	)
	t.Run(
		"AMoveThroughTheIndexAgainstADeleteLeavesNoRecord",
		func(t *testing.T) { aMoveThroughTheIndexAgainstADeleteLeavesNoRecord(t, open) },
	)
	t.Run(
		"ConcurrentInsertsOfOneAbsentKeyLeaveOneWinner",
		func(t *testing.T) { concurrentInsertsOfOneAbsentKeyLeaveOneWinner(t, open) },
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

func concurrentMovesOfTheFirstIndexEntryMoveEachRecordOnce(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	first, second := openTwoHandles(t, open)
	handles := []rankedRecords{openRankedRecords(t, first), openRankedRecords(t, second)}
	keys := numberedWords(recordsToMove)
	handles[0].write(t, func(tx *vault.Txn) error {
		for rank, key := range keys {
			if _, err := handles[0].records.Put(tx, key, ranked{rank: int64(rank)}); err != nil {
				return wrapTest(err)
			}
		}

		return nil
	})

	movedByWorker := make([][]string, concurrentWriters)
	work := make([]func() error, concurrentWriters)
	for worker := range work {
		work[worker] = func() error {
			for {
				key, moved, err := handles[worker%len(handles)].moveFirstUnmoved()
				if err != nil || !moved {
					return err
				}
				movedByWorker[worker] = append(movedByWorker[worker], key)
			}
		}
	}
	if err := failuresOfConcurrent(work); err != nil {
		t.Fatalf("Update: %v", err)
	}

	moved := slices.Concat(movedByWorker...)
	slices.Sort(moved)
	if !slices.Equal(moved, keys) {
		t.Fatalf("moved = %v, want each of %v once", moved, keys)
	}
}

type rankedRecords struct {
	vault   *vault.Vault
	records *vault.Collection[string, ranked]
	byRank  *vault.ValueIndex[rankEntry, rankEntry]
}

type ranked struct {
	rank    int64
	changes int
}

type rankEntry struct {
	rank int64
	key  string
}

var rankEntryLayout = vault.PairKey(vault.IntegerKeyPart, vault.TextKeyPart).KeyLayoutFor(
	func(entry rankEntry) (int64, string) { return entry.rank, entry.key },
	func(rank int64, key string) rankEntry { return rankEntry{rank: rank, key: key} },
)

func openRankedRecords(t *testing.T, v *vault.Vault) rankedRecords {
	t.Helper()

	byRank, err := v.RegisterValueIndex(
		vault.Name("ranked_by_rank"),
		"1",
		rankEntryLayout,
		func(entry rankEntry) rankEntry { return entry },
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	records, err := v.RegisterProjectedCollection(
		context.Background(),
		vault.Name("ranked"),
		stringKeyLayout,
		rankedCodec{},
		vault.ProjectedAs(
			func(key string, record ranked) rankEntry { return rankEntry{rank: record.rank, key: key} },
			byRank,
		),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}

	return rankedRecords{vault: v, records: records, byRank: byRank}
}

type rankedCodec struct{}

func (rankedCodec) Encode(record ranked) ([]byte, error) {
	return fmt.Appendf(nil, "%d/%d", record.rank, record.changes), nil
}

func (rankedCodec) Decode(raw []byte) (ranked, error) {
	var record ranked
	if _, err := fmt.Sscanf(string(raw), "%d/%d", &record.rank, &record.changes); err != nil {
		return ranked{}, fmt.Errorf("decode ranked record %q: %w", raw, err)
	}

	return record, nil
}

func (r rankedRecords) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := r.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (r rankedRecords) moveFirstUnmoved() (key string, moved bool, err error) {
	err = r.vault.Update(context.Background(), func(tx *vault.Txn) error {
		key, moved = "", false
		first, found, err := r.firstEntryOf(tx)
		if err != nil || !found || first.rank >= movedRank {
			return err
		}
		key, moved = first.key, true

		return r.rerank(tx, first.key, movedRank)
	})

	return key, moved, err
}

func (r rankedRecords) firstEntryOf(tx *vault.Txn) (rankEntry, bool, error) {
	var first rankEntry

	var found bool
	if err := r.byRank.Scan(tx, vault.EveryKey(), func(entry rankEntry) (bool, error) {
		first, found = entry, true

		return false, nil
	}); err != nil {
		return rankEntry{}, false, wrapTest(err)
	}

	return first, found, nil
}

func (r rankedRecords) rerank(tx *vault.Txn, key string, rank int64) error {
	record, found, err := r.records.Get(tx, key)
	if err != nil {
		return wrapTest(err)
	}
	if !found {
		return nil
	}
	if _, err := r.records.Put(
		tx,
		key,
		ranked{rank: rank, changes: record.changes + 1},
	); err != nil {
		return wrapTest(err)
	}

	return nil
}

func movesThroughTheIndexAndByKeyKeepEveryChange(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	first, second := openTwoHandles(t, open)
	throughIndex, byKey := openRankedRecords(t, first), openRankedRecords(t, second)
	throughIndex.write(t, func(tx *vault.Txn) error {
		_, err := throughIndex.records.Put(tx, contendedKey, ranked{})

		return err
	})

	work := make([]func() error, 0, concurrentWriters)
	for range concurrentWriters / 2 {
		work = append(
			work,
			func() error { return repeated(throughIndex.moveFirst) },
			func() error { return repeated(byKey.moveContended) },
		)
	}
	if err := failuresOfConcurrent(work); err != nil {
		t.Fatalf("Update: %v", err)
	}

	record := throughIndex.recordAt(t, contendedKey)
	if record.changes != concurrentWriters*incrementsPerWriter {
		t.Fatalf("changes = %d, want %d", record.changes, concurrentWriters*incrementsPerWriter)
	}
	want := []rankEntry{{rank: record.rank, key: contendedKey}}
	if got := throughIndex.entries(t); !slices.Equal(got, want) {
		t.Fatalf("index entries = %v, want %v", got, want)
	}
}

func (r rankedRecords) moveFirst() error {
	return r.vault.Update(context.Background(), func(tx *vault.Txn) error {
		first, found, err := r.firstEntryOf(tx)
		if err != nil || !found {
			return err
		}
		record, _, err := r.records.Get(tx, first.key)
		if err != nil {
			return wrapTest(err)
		}

		return r.rerank(tx, first.key, movedRank+int64(record.changes))
	})
}

func (r rankedRecords) moveContended() error {
	return r.vault.Update(context.Background(), func(tx *vault.Txn) error {
		record, _, err := r.records.Get(tx, contendedKey)
		if err != nil {
			return wrapTest(err)
		}

		return r.rerank(tx, contendedKey, int64(record.changes))
	})
}

func (r rankedRecords) recordAt(t *testing.T, key string) ranked {
	t.Helper()

	var record ranked
	if err := r.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		record, _, err = r.records.Get(tx, key)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return record
}

func (r rankedRecords) entries(t *testing.T) []rankEntry {
	t.Helper()

	var entries []rankEntry
	if err := r.vault.View(context.Background(), func(tx *vault.Txn) error {
		return r.byRank.Scan(tx, vault.EveryKey(), func(entry rankEntry) (bool, error) {
			entries = append(entries, entry)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return entries
}

func aMoveThroughTheIndexAgainstADeleteLeavesNoRecord(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	first, second := openTwoHandles(t, open)
	throughIndex, byKey := openRankedRecords(t, first), openRankedRecords(t, second)

	for round := range contendedRounds {
		key := strconv.Itoa(round)
		throughIndex.write(t, func(tx *vault.Txn) error {
			_, err := throughIndex.records.Put(tx, key, ranked{})

			return err
		})
		if err := failuresOfConcurrent([]func() error{
			func() error {
				_, _, err := throughIndex.moveFirstUnmoved()

				return err
			},
			func() error { return byKey.deleteRecord(key) },
		}); err != nil {
			t.Fatalf("Update: %v", err)
		}
	}

	if got := throughIndex.entries(t); len(got) != 0 {
		t.Fatalf("index entries after deletes = %v, want none", got)
	}
	if length := throughIndex.length(t); length != 0 {
		t.Fatalf("records after deletes = %d, want 0", length)
	}
}

func (r rankedRecords) deleteRecord(key string) error {
	return r.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := r.records.Delete(tx, key)

		return err
	})
}

func (r rankedRecords) length(t *testing.T) int {
	t.Helper()

	var length int
	if err := r.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		length, err = r.records.Len(tx)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return length
}

func concurrentInsertsOfOneAbsentKeyLeaveOneWinner(
	t *testing.T,
	open func(int64) (vault.Engine, error),
) {
	first, second := openTwoHandles(t, open)
	handles := []*vault.Vault{first, second}
	words := []*vault.Collection[string, string]{
		register(t, first, "words"),
		register(t, second, "words"),
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
							words[writer%len(words)],
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
		if stored := storedWord(t, first, words[0], key); stored != strconv.Itoa(winner) {
			t.Fatalf("round %d stored %q, want winner %d", round, stored, winner)
		}
	}
}

func insertAbsent(
	tx *vault.Txn,
	words *vault.Collection[string, string],
	key, value string,
) (bool, error) {
	_, found, err := words.Get(tx, key)
	if err != nil || found {
		return false, err
	}
	if _, err := words.Put(tx, key, value); err != nil {
		return false, wrapTest(err)
	}

	return true, nil
}

func storedWord(
	t *testing.T,
	v *vault.Vault,
	words *vault.Collection[string, string],
	key string,
) string {
	t.Helper()

	var value string
	if err := v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		value, _, err = words.Get(tx, key)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return value
}
