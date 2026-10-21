package retainedrecords_test

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
)

const (
	bucket        = "words"
	wordRetention = time.Hour
)

var wordKeys = vault.SingleKey(vault.TextKeyPart).KeyLayout()

var (
	writtenAt         = time.UnixMilli(1_700_000_000_000)
	errEngineRefusing = errors.New("engine refuses writes")
)

type textCodec struct{}

func (textCodec) Encode(text string) ([]byte, error) { return []byte(text), nil }
func (textCodec) Decode(raw []byte) (string, error)  { return string(raw), nil }

type sweepClock struct {
	now time.Time
}

func (clock sweepClock) Now() time.Time { return clock.now }

func (sweepClock) Sleep(context.Context, time.Duration) error { return context.Canceled }

type recordingObserver struct {
	forgotten    map[string]int
	notForgotten []error
}

func (o *recordingObserver) ExpiredRecordsForgotten(_ context.Context, bucket string, amount int) {
	o.forgotten[bucket] += amount
}

func (o *recordingObserver) ExpiredRecordsNotForgotten(_ context.Context, _ string, cause error) {
	o.notForgotten = append(o.notForgotten, cause)
}

type refusingEngine struct {
	vault.Engine
	refusing *bool
}

func (engine refusingEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if *engine.refusing {
		return errEngineRefusing
	}
	return engine.Engine.Update(ctx, fn)
}

type scene struct {
	storage   *vault.Vault
	retention *retainedrecords.Retention
	words     *retainedrecords.Records[string, string]
	byValue   *vault.ValueIndex[retainedrecords.Record[string, string], string]
	observer  *recordingObserver
	refusing  *bool
}

func sceneSweepingAt(t *testing.T, now time.Time) scene {
	t.Helper()
	refusing := false
	storage, err := vault.New(
		refusingEngine{Engine: memoryvault.OpenEngine(0), refusing: &refusing},
		nil,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	observer := &recordingObserver{forgotten: map[string]int{}}
	retention := retainedrecords.New(storage, sweepClock{now: now}, observer)
	byValue, err := storage.RegisterValueIndex(
		bucket+"_by_value",
		"1",
		wordKeys,
		func(record retainedrecords.Record[string, string]) string { return record.Value },
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	words, err := retention.Register(
		t.Context(),
		bucket,
		wordKeys,
		textCodec{},
		wordRetention,
		byValue,
	)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return scene{
		storage:   storage,
		retention: retention,
		words:     words,
		byValue:   byValue,
		observer:  observer,
		refusing:  &refusing,
	}
}

func (s scene) write(t *testing.T, key string, at time.Time) {
	t.Helper()
	if err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		return s.words.Put(tx, key, "value of "+key, at)
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (s scene) foundAt(t *testing.T, key string, at time.Time) bool {
	t.Helper()
	var found bool
	if err := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		var err error
		_, found, err = s.words.Get(tx, key, at)
		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	return found
}

func (s scene) storedRecords(t *testing.T) int {
	t.Helper()
	counts, err := s.storage.RecordCountsByBucket(t.Context())
	if err != nil {
		t.Fatalf("RecordCountsByBucket: %v", err)
	}
	return counts[bucket]
}

func (s scene) indexedValues(t *testing.T) []string {
	t.Helper()
	var values []string
	if err := s.storage.View(t.Context(), func(tx *vault.Txn) error {
		return s.byValue.Scan(tx, vault.EveryKey(), func(value string) (bool, error) {
			values = append(values, value)
			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}
	return values
}

func TestARecordIsFoundUntilItsRetentionEnds(t *testing.T) {
	s := sceneSweepingAt(t, writtenAt)
	s.write(t, "a", writtenAt)

	if !s.foundAt(t, "a", writtenAt.Add(wordRetention-time.Millisecond)) {
		t.Fatal("a record within its retention is absent")
	}
	if s.foundAt(t, "a", writtenAt.Add(wordRetention)) {
		t.Fatal("a record past its retention is found")
	}
}

func TestAWriteRenewsTheRetention(t *testing.T) {
	s := sceneSweepingAt(t, writtenAt)
	s.write(t, "a", writtenAt)
	s.write(t, "a", writtenAt.Add(wordRetention/2))

	if !s.foundAt(t, "a", writtenAt.Add(wordRetention)) {
		t.Fatal("a rewritten record ends with its first retention")
	}
}

func TestASweepForgetsOnlyExpiredRecords(t *testing.T) {
	s := sceneSweepingAt(t, writtenAt.Add(wordRetention))
	s.write(t, "expired", writtenAt)
	s.write(t, "kept", writtenAt.Add(time.Minute))

	if err := s.retention.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if s.storedRecords(t) != 1 || !s.foundAt(t, "kept", writtenAt.Add(wordRetention)) {
		t.Fatalf("stored %d records, want only the kept one", s.storedRecords(t))
	}
	if s.observer.forgotten[bucket] != 1 {
		t.Fatalf("forgotten %v, want one %s record", s.observer.forgotten, bucket)
	}
}

func TestASweepForgetsTheIndexEntriesOfExpiredRecords(t *testing.T) {
	s := sceneSweepingAt(t, writtenAt.Add(wordRetention))
	s.write(t, "expired", writtenAt)
	s.write(t, "kept", writtenAt.Add(time.Minute))

	if err := s.retention.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := s.indexedValues(t); !slices.Equal(got, []string{"value of kept"}) {
		t.Fatalf("indexed %v, want only the value of the kept record", got)
	}
}

func TestASweepForgetsMoreExpiredRecordsThanOneBatch(t *testing.T) {
	s := sceneSweepingAt(t, writtenAt.Add(wordRetention))
	const expired = 1001
	if err := s.storage.Update(t.Context(), func(tx *vault.Txn) error {
		for key := range expired {
			if err := s.words.Put(tx, strconv.Itoa(key), "value", writtenAt); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := s.retention.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if s.storedRecords(t) != 0 || s.observer.forgotten[bucket] != expired {
		t.Fatalf(
			"stored %d records and forgot %d, want none stored and %d forgotten",
			s.storedRecords(t), s.observer.forgotten[bucket], expired,
		)
	}
}

func TestASweepTheStorageRefusesIsReported(t *testing.T) {
	s := sceneSweepingAt(t, writtenAt.Add(wordRetention))
	s.write(t, "expired", writtenAt)
	*s.refusing = true

	if err := s.retention.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(s.observer.notForgotten) != 1 ||
		!errors.Is(s.observer.notForgotten[0], errEngineRefusing) {
		t.Fatalf("not forgotten %v, want the refusal", s.observer.notForgotten)
	}
	if len(s.observer.forgotten) != 0 {
		t.Fatalf("forgotten %v, want nothing", s.observer.forgotten)
	}
}
