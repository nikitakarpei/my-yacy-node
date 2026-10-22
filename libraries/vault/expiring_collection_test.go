package vault_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
)

const wordLifetime = 32 * time.Hour

const unreadableMeaning = "unreadable"

type expiringDictionary struct {
	vault          *vault.Vault
	clock          *fakeClock
	words          *vault.ExpiringCollection[string, string]
	byMeaning      *vault.ValueIndex[definition, definition]
	removals       *removalReports
	firstPassPause time.Duration
}

type removalReports struct {
	passesBegan    int
	passesFinished int
	forgotten      int
	notForgotten   []error
}

func (r *removalReports) ExpiryPassBegan(context.Context, vault.Name, time.Time) {
	r.passesBegan++
}

func (r *removalReports) ExpiryPassFinished(context.Context, vault.Name, time.Time) {
	r.passesFinished++
}

func (r *removalReports) ExpiredRecordsForgotten(_ context.Context, _ vault.Name, amount int) {
	r.forgotten += amount
}

func (r *removalReports) ExpiredRecordsNotForgotten(_ context.Context, _ vault.Name, cause error) {
	r.notForgotten = append(r.notForgotten, cause)
}

type meaningRefusingCodec struct{}

var errUnreadableMeaning = errors.New("unreadable meaning")

func (meaningRefusingCodec) Encode(meaning string) ([]byte, error) { return []byte(meaning), nil }

func (meaningRefusingCodec) Decode(raw []byte) (string, error) {
	if string(raw) == unreadableMeaning {
		return "", errUnreadableMeaning
	}

	return string(raw), nil
}

var errWritesRefused = errors.New("writes refused")

type refusingWritesEngine struct {
	vault.Engine
	refusing atomic.Bool
}

func (e *refusingWritesEngine) Update(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if e.refusing.Load() {
		return errWritesRefused
	}

	return e.Engine.Update(ctx, fn)
}

func openExpiringDictionary(
	t *testing.T,
	engine vault.Engine,
	version string,
	meaningOf func(string) string,
) expiringDictionary {
	t.Helper()

	clock := newFakeClock(testEpoch)
	v, err := vault.New(engine, nil, clock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	byMeaning, err := v.RegisterValueIndex(
		vault.Name("words_by_meaning"),
		version,
		definitionEntries,
		func(record definition) definition {
			return definition{word: record.word, meaning: meaningOf(record.meaning)}
		},
	)
	if err != nil {
		t.Fatalf("RegisterValueIndex: %v", err)
	}
	removals := &removalReports{}
	words, err := v.RegisterExpiringCollection(
		context.Background(),
		vault.Name("words"),
		stringKeyLayout,
		meaningRefusingCodec{},
		wordLifetime,
		removals,
		vault.ProjectedAs(definitionOf, byMeaning),
	)
	if err != nil {
		t.Fatalf("RegisterExpiringCollection: %v", err)
	}

	return expiringDictionary{
		vault:          v,
		clock:          clock,
		words:          words,
		byMeaning:      byMeaning,
		removals:       removals,
		firstPassPause: clock.sleepAfterPass(),
	}
}

func (d expiringDictionary) define(t *testing.T, definitions ...definition) {
	t.Helper()

	if err := d.vault.Update(context.Background(), func(tx *vault.Txn) error {
		for _, defined := range definitions {
			if err := d.words.Put(tx, defined.word, defined.meaning); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (d expiringDictionary) meaningOf(t *testing.T, word string) (string, bool) {
	t.Helper()

	var meaning string
	var found bool
	if err := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		meaning, found, err = d.words.Get(tx, word)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return meaning, found
}

func (d expiringDictionary) entries(t *testing.T) []string {
	t.Helper()

	var entries []string
	if err := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		return d.byMeaning.Scan(tx, vault.EveryKey(), func(entry definition) (bool, error) {
			entries = append(entries, entry.meaning+"/"+entry.word)

			return true, nil
		})
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return entries
}

func (d expiringDictionary) runRemovalPass() {
	d.clock.wakeForPass()
	d.clock.sleepAfterPass()
}

func TestAnExpiringRecordReadsAsAbsentOnceItsLifetimeEnds(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("a", "x"))

	d.clock.advanceBy(wordLifetime - time.Second)
	if meaning, found := d.meaningOf(t, "a"); !found || meaning != "x" {
		t.Fatalf("before the lifetime ends: Get = %q, %v, want x, true", meaning, found)
	}
	d.clock.advanceBy(time.Second)
	if _, found := d.meaningOf(t, "a"); found {
		t.Fatal("once the lifetime ends: Get found the record, want absent")
	}
}

func TestAPutRenewsTheLifetimeOfAnExpiringRecord(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("b", "x"))

	d.clock.advanceBy(wordLifetime - time.Second)
	d.define(t, definitionOf("b", "x"))
	d.clock.advanceBy(wordLifetime - time.Second)

	if _, found := d.meaningOf(t, "b"); !found {
		t.Fatal("Get = absent, want the renewed record")
	}
}

func TestARepeatedWriteTakesTheTimeOfItsOwnRun(t *testing.T) {
	d := openExpiringDictionary(
		t,
		vaultenginetest.EngineRepeatingWrites(newDoubleEngine()),
		firstVersion,
		plainMeaning,
	)
	runs := 0
	if err := d.vault.Update(context.Background(), func(tx *vault.Txn) error {
		runs++
		if runs == 1 {
			d.clock.advanceBy(wordLifetime)
		}

		return d.words.Put(tx, "a", "x")
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	d.clock.advanceBy(wordLifetime - time.Second)
	if _, found := d.meaningOf(t, "a"); !found {
		t.Fatal("Get = absent, want the record the second run wrote")
	}
}

func TestRemovalPassesRepeatEveryThirtySecondOfTheLifetime(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	if want := wordLifetime / 32; d.firstPassPause != want {
		t.Fatalf("pause after a pass = %s, want %s", d.firstPassPause, want)
	}
}

func TestRemovalForgetsExpiredRecordsWithTheirIndexEntries(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("a", "x"))
	d.clock.advanceBy(wordLifetime / 2)
	d.define(t, definitionOf("b", "y"))
	d.clock.advanceBy(wordLifetime / 2)

	d.runRemovalPass()

	if got, want := d.entries(t), []string{"y/b"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	if d.removals.forgotten != 1 {
		t.Fatalf("forgotten = %d, want 1", d.removals.forgotten)
	}
	if d.removals.passesBegan != 2 || d.removals.passesFinished != 2 {
		t.Fatalf(
			"passes began %d, finished %d, want 2 and 2",
			d.removals.passesBegan,
			d.removals.passesFinished,
		)
	}
}

func TestOnePassForgetsMoreExpiredRecordsThanOneChunk(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	const amountExpired = 2500
	definitions := make([]definition, 0, amountExpired)
	for position := range amountExpired {
		definitions = append(definitions, definitionOf(fmt.Sprintf("word %04d", position), "x"))
	}
	d.define(t, definitions...)
	d.clock.advanceBy(wordLifetime)

	d.runRemovalPass()

	if entries := d.entries(t); len(entries) != 0 {
		t.Fatalf("%d entries left, want none", len(entries))
	}
	if d.removals.forgotten != amountExpired {
		t.Fatalf("forgotten = %d, want %d", d.removals.forgotten, amountExpired)
	}
}

func TestARemovalPassTheStorageRefusesIsReportedAndLeftUnfinished(t *testing.T) {
	engine := &refusingWritesEngine{Engine: newDoubleEngine()}
	d := openExpiringDictionary(t, engine, firstVersion, plainMeaning)
	d.define(t, definitionOf("a", "x"))
	d.clock.advanceBy(wordLifetime)
	engine.refusing.Store(true)

	d.runRemovalPass()

	if len(d.removals.notForgotten) != 1 ||
		!errors.Is(d.removals.notForgotten[0], errWritesRefused) {
		t.Fatalf("not forgotten = %v, want the refusal", d.removals.notForgotten)
	}
	if d.removals.passesFinished != 1 {
		t.Fatalf("passes finished = %d, want only the first", d.removals.passesFinished)
	}
}

func TestAPutOverAnExpiredStoredRecordReplacesItsIndexEntry(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("a", "x"))
	d.clock.advanceBy(wordLifetime)

	d.define(t, definitionOf("a", "y"))

	if got, want := d.entries(t), []string{"y/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestRemovalSkipsAndReportsARecordItCannotDecodeOnEveryPass(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)
	d.define(t, definitionOf("a", unreadableMeaning), definitionOf("b", "y"))
	d.clock.advanceBy(wordLifetime)

	d.runRemovalPass()
	d.runRemovalPass()

	if got, want := d.entries(t), []string{unreadableMeaning + "/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	if d.removals.forgotten != 1 {
		t.Fatalf("forgotten = %d, want 1", d.removals.forgotten)
	}
	if len(d.removals.notForgotten) != 2 ||
		!errors.Is(d.removals.notForgotten[1], errUnreadableMeaning) {
		t.Fatalf("not forgotten = %v, want the unreadable record twice", d.removals.notForgotten)
	}
}

func TestAStaleIndexOfAnExpiringCollectionRebuildsFromStoredRecords(t *testing.T) {
	engine := newDoubleEngine()
	earlier := openExpiringDictionary(t, engine, firstVersion, plainMeaning)
	earlier.define(t, definitionOf("a", "x"))

	d := openExpiringDictionary(t, engine, secondVersion, strings.ToUpper)

	if got, want := d.entries(t), []string{"X/a"}; !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestAnExpiringPutInsideAViewIsRefused(t *testing.T) {
	d := openExpiringDictionary(t, newDoubleEngine(), firstVersion, plainMeaning)

	if err := d.vault.View(context.Background(), func(tx *vault.Txn) error {
		return d.words.Put(tx, "a", "x")
	}); err == nil {
		t.Fatal("Put inside View succeeded, want error")
	}
}

func TestRegistrationRefusesAnExpiringCollectionItCannotRemove(t *testing.T) {
	for name, registration := range map[string]struct {
		lifetime time.Duration
		observer vault.ExpiryObserver
	}{
		"lifetime under one second": {time.Second - 1, &removalReports{}},
		"no expiry observer":        {wordLifetime, nil},
	} {
		v, err := openDouble()
		if err != nil {
			t.Fatalf("openDouble: %v", err)
		}
		if _, err := v.RegisterExpiringCollection(
			context.Background(),
			vault.Name("words"),
			stringKeyLayout,
			stringValueCodec{},
			registration.lifetime,
			registration.observer,
			nil,
		); err == nil {
			t.Errorf("%s: registration succeeded, want error", name)
		}
	}
}
